package dao

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	reminderConsumerName         = "be-library-reminder"
	preferenceReconcileBatchSize = 200
)

type PreferenceChange struct {
	Revision  int64
	StudentID string
	Enabled   bool
}

type ReminderStatusCount struct {
	Type   string
	Status string
	Count  int64
}

type ReminderMetricsSnapshot struct {
	Jobs           []ReminderStatusCount
	Outbox         []ReminderStatusCount
	OldestDueJobAt *time.Time
	OldestOutboxAt *time.Time
	ActiveUsers    int64
}

type ReminderDAO struct{ db *gorm.DB }

func NewReminderDAO(db *gorm.DB) *ReminderDAO { return &ReminderDAO{db: db} }

// RecoverOrphanedWork 在单实例启动时恢复上次进程中断留下的处理中任务。
func (d *ReminderDAO) RecoverOrphanedWork(ctx context.Context) error {
	return d.recoverClaimedWork(ctx, nil)
}

// RecoverStaleWork 周期性恢复超过 claim 时限的处理中任务。
func (d *ReminderDAO) RecoverStaleWork(ctx context.Context, before time.Time) error {
	return d.recoverClaimedWork(ctx, &before)
}

func (d *ReminderDAO) recoverClaimedWork(ctx context.Context, before *time.Time) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		jobs := tx.Model(&NotificationJob{}).Where("status = ?", JobRunning)
		if before != nil {
			jobs = jobs.Where("updated_at < ?", *before)
		}
		if err := jobs.Updates(map[string]any{
			"status":   JobPending,
			"attempts": gorm.Expr("CASE WHEN attempts > 0 THEN attempts - 1 ELSE 0 END"),
			"version":  gorm.Expr("version + 1"),
		}).Error; err != nil {
			return err
		}
		outbox := tx.Model(&NotificationOutbox{}).Where("status = ?", OutboxSending)
		if before != nil {
			outbox = outbox.Where("updated_at < ?", *before)
		}
		// sending 可能已完成远程投递，不撤销 attempts；Feed 的去重键保证重试安全。
		return outbox.Updates(map[string]any{"status": OutboxFailed, "last_error": "sending claim timed out"}).Error
	})
}

// Transaction 用于原子应用完整的上游快照。回调接收绑定事务的 DAO，
// 因此现有 DAO 方法无需暴露底层 GORM 句柄。
func (d *ReminderDAO) Transaction(ctx context.Context, fn func(*ReminderDAO) error) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&ReminderDAO{db: tx})
	})
}

func (d *ReminderDAO) Cursor(ctx context.Context) (int64, error) {
	var cursor LibraryPreferenceSyncCursor
	err := d.db.WithContext(ctx).Where("consumer_name = ?", reminderConsumerName).First(&cursor).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return -1, nil
	}
	return cursor.LastRevision, err
}

// 使用事务存储本地的提醒开关
func (d *ReminderDAO) ApplyPreferenceChanges(ctx context.Context, changes []PreferenceChange, nextRevision int64) ([]LibraryReminderSubscription, error) {
	var enabled []LibraryReminderSubscription
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, change := range changes {
			var sub LibraryReminderSubscription
			err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("student_id = ?", change.StudentID).First(&sub).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				sub = LibraryReminderSubscription{StudentID: change.StudentID, Enabled: change.Enabled, FeedRevision: change.Revision, PreferenceVersion: 1, AuthStatus: SubscriptionAuthUnknown}
				if err := tx.Create(&sub).Error; err != nil {
					return err
				}
				if change.Enabled {
					enabled = append(enabled, sub)
				}
				continue
			}
			if err != nil {
				return err
			}
			if change.Revision <= sub.FeedRevision {
				continue
			}
			wasEnabled := sub.Enabled
			updates := map[string]any{"feed_revision": change.Revision}
			if wasEnabled != change.Enabled {
				updates["enabled"] = change.Enabled
				updates["preference_version"] = gorm.Expr("preference_version + 1")
				updates["baseline_completed"] = false
			}
			if err := tx.Model(&sub).Updates(updates).Error; err != nil {
				return err
			}
			if !change.Enabled {
				if err := tx.Model(&NotificationJob{}).Where("student_id = ? AND status IN ?", change.StudentID, []string{JobPending, JobRunning}).Updates(map[string]any{"status": JobCancelled, "version": gorm.Expr("version + 1")}).Error; err != nil {
					return err
				}
				if err := tx.Model(&NotificationOutbox{}).Where("student_id = ? AND status IN ?", change.StudentID, []string{OutboxPending, OutboxSending, OutboxFailed}).Update("status", OutboxSuppressed).Error; err != nil {
					return err
				}
			} else if !wasEnabled {
				if err := tx.Where("student_id = ?", change.StudentID).First(&sub).Error; err != nil {
					return err
				}
				enabled = append(enabled, sub)
			}
		}
		cursor := LibraryPreferenceSyncCursor{ConsumerName: reminderConsumerName, LastRevision: nextRevision}
		return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "consumer_name"}}, DoUpdates: clause.AssignmentColumns([]string{"last_revision", "updated_at"})}).Create(&cursor).Error
	})
	return enabled, err
}

// 重建通知开关信息
func (d *ReminderDAO) RebuildPreferences(ctx context.Context, users []PreferenceChange, revision int64) ([]LibraryReminderSubscription, error) {
	return d.reconcilePreferences(ctx, users, revision, true)
}

func (d *ReminderDAO) CalibratePreferences(ctx context.Context, users []PreferenceChange) ([]LibraryReminderSubscription, error) {
	return d.reconcilePreferences(ctx, users, 0, false)
}

func (d *ReminderDAO) reconcilePreferences(ctx context.Context, users []PreferenceChange, revision int64, updateCursor bool) ([]LibraryReminderSubscription, error) {
	enabled := make([]LibraryReminderSubscription, 0, len(users))
	// 从 Feed 获取的权威数据
	authoritative := make(map[string]int64, len(users))
	for _, user := range users {
		if current, exists := authoritative[user.StudentID]; !exists || user.Revision > current {
			authoritative[user.StudentID] = user.Revision
		}
	}
	// 按主键分批加锁并提交，避免校准期间长时间锁住整张订阅表。
	var afterID int64
	for {
		var existing []LibraryReminderSubscription
		var batchEnabled []LibraryReminderSubscription
		err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("id > ?", afterID).Order("id ASC").Limit(preferenceReconcileBatchSize).
				Find(&existing).Error; err != nil {
				return err
			}
			for i := range existing {
				sub := existing[i]
				feedRevision, shouldEnable := authoritative[sub.StudentID]
				if shouldEnable {
					// 全量快照可能跨过“关闭后重开”，推进版本前必须让旧基线和任务失效。
					resetBaseline := !sub.Enabled || feedRevision > sub.FeedRevision
					updates := map[string]any{}
					if feedRevision > sub.FeedRevision {
						updates["feed_revision"] = feedRevision
					}
					if resetBaseline {
						updates["enabled"] = true
						updates["baseline_completed"] = false
						updates["preference_version"] = gorm.Expr("preference_version + 1")
					}
					if len(updates) > 0 {
						if err := tx.Model(&sub).Updates(updates).Error; err != nil {
							return err
						}
					}
					if resetBaseline {
						if err := suppressStudentWorkTx(tx, sub.StudentID); err != nil {
							return err
						}
						if err := tx.Where("id = ?", sub.ID).First(&sub).Error; err != nil {
							return err
						}
						batchEnabled = append(batchEnabled, sub)
					}
					continue
				}
				if !sub.Enabled {
					continue
				}
				if err := tx.Model(&sub).Updates(map[string]any{"enabled": false, "baseline_completed": false, "preference_version": gorm.Expr("preference_version + 1")}).Error; err != nil {
					return err
				}
				if err := suppressStudentWorkTx(tx, sub.StudentID); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return enabled, err
		}
		if len(existing) == 0 {
			break
		}
		for i := range existing {
			delete(authoritative, existing[i].StudentID)
		}
		enabled = append(enabled, batchEnabled...)
		afterID = existing[len(existing)-1].ID
	}

	// 新增订阅也分批提交，限制单次事务时长和 SQL 体积。
	missing := make([]LibraryReminderSubscription, 0, preferenceReconcileBatchSize)
	createMissing := func() error {
		if len(missing) == 0 {
			return nil
		}
		if err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			return tx.Create(&missing).Error
		}); err != nil {
			return err
		}
		enabled = append(enabled, missing...)
		missing = missing[:0]
		return nil
	}
	for studentID, feedRevision := range authoritative {
		missing = append(missing, LibraryReminderSubscription{StudentID: studentID, Enabled: true, FeedRevision: feedRevision, PreferenceVersion: 1, AuthStatus: SubscriptionAuthUnknown})
		if len(missing) == preferenceReconcileBatchSize {
			if err := createMissing(); err != nil {
				return enabled, err
			}
		}
	}
	if err := createMissing(); err != nil {
		return enabled, err
	}
	if !updateCursor {
		return enabled, nil
	}
	cursor := LibraryPreferenceSyncCursor{ConsumerName: reminderConsumerName, LastRevision: revision}
	err := d.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "consumer_name"}}, DoUpdates: clause.AssignmentColumns([]string{"last_revision", "updated_at"})}).Create(&cursor).Error
	return enabled, err
}

func suppressStudentWorkTx(tx *gorm.DB, studentID string) error {
	if err := tx.Model(&NotificationJob{}).Where("student_id = ? AND status IN ?", studentID, []string{JobPending, JobRunning}).Updates(map[string]any{"status": JobCancelled, "version": gorm.Expr("version + 1")}).Error; err != nil {
		return err
	}
	return tx.Model(&NotificationOutbox{}).Where("student_id = ? AND status IN ?", studentID, []string{OutboxPending, OutboxSending, OutboxFailed}).Update("status", OutboxSuppressed).Error
}

func (d *ReminderDAO) EnabledSubscriptions(ctx context.Context, limit int) ([]LibraryReminderSubscription, error) {
	var rows []LibraryReminderSubscription
	err := d.db.WithContext(ctx).Where("enabled = ?", true).
		Order("last_full_refresh_at IS NULL DESC, last_full_refresh_at ASC, id ASC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (d *ReminderDAO) PendingBaselineSubscriptions(ctx context.Context, limit int) ([]LibraryReminderSubscription, error) {
	var rows []LibraryReminderSubscription
	err := d.db.WithContext(ctx).Where("enabled = ? AND baseline_completed = ?", true, false).Order("id ASC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (d *ReminderDAO) ActiveSubscriptions(ctx context.Context, now time.Time, limit int) ([]LibraryReminderSubscription, error) {
	var rows []LibraryReminderSubscription
	err := d.db.WithContext(ctx).Model(&LibraryReminderSubscription{}).
		Distinct("library_reminder_subscriptions.*").
		Joins("JOIN reservation_snapshots r ON r.student_id = library_reminder_subscriptions.student_id").
		Where("library_reminder_subscriptions.enabled = ? AND r.start_at <= ? AND r.end_at > ? AND UPPER(r.status) NOT IN ?", true, now, now, []string{"CANCEL", "STOP", "FINISH", "LEAVE_EARLY", "MISS"}).
		Order("library_reminder_subscriptions.last_active_scan_at IS NULL DESC, library_reminder_subscriptions.last_active_scan_at ASC, library_reminder_subscriptions.id ASC").
		Limit(limit).Find(&rows).Error
	return rows, err
}

func (d *ReminderDAO) MetricsSnapshot(ctx context.Context, now time.Time, maxAttempts int) (ReminderMetricsSnapshot, error) {
	var snapshot ReminderMetricsSnapshot
	if err := d.db.WithContext(ctx).Model(&NotificationJob{}).
		Select("type, status, COUNT(*) AS count").Group("type, status").Scan(&snapshot.Jobs).Error; err != nil {
		return snapshot, err
	}
	if err := d.db.WithContext(ctx).Model(&NotificationOutbox{}).
		Select("'' AS type, status, COUNT(*) AS count").Group("status").Scan(&snapshot.Outbox).Error; err != nil {
		return snapshot, err
	}
	var oldestJob NotificationJob
	err := d.db.WithContext(ctx).Where("status = ? AND run_at <= ?", JobPending, now).Order("run_at ASC, id ASC").First(&oldestJob).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return snapshot, err
	}
	if err == nil {
		runAt := oldestJob.RunAt
		snapshot.OldestDueJobAt = &runAt
	}
	var oldestOutbox NotificationOutbox
	// 与 ClaimOutbox 的领取条件保持一致，只统计仍可被领取的记录，避免已耗尽重试的 failed 记录永久抬高指标。
	err = d.db.WithContext(ctx).Where("status IN ? AND next_attempt_at <= ? AND attempts < ?", []string{OutboxPending, OutboxFailed}, now, maxAttempts).Order("created_at ASC, id ASC").First(&oldestOutbox).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return snapshot, err
	}
	if err == nil {
		createdAt := oldestOutbox.CreatedAt
		snapshot.OldestOutboxAt = &createdAt
	}
	if err := d.db.WithContext(ctx).Model(&ReservationSnapshot{}).
		Distinct("student_id").Where("start_at <= ? AND end_at > ? AND UPPER(status) NOT IN ?", now, now, []string{"CANCEL", "STOP", "FINISH", "LEAVE_EARLY", "MISS"}).Count(&snapshot.ActiveUsers).Error; err != nil {
		return snapshot, err
	}
	return snapshot, nil
}

const historyCleanupBatchSize = 500

// CleanupHistory 分批删除超过保留期的终态任务与 outbox 记录，避免历史表无限增长拖慢周期聚合和投递。
// failed 且已耗尽重试的 outbox 记录同样视为终态一并清理。
func (d *ReminderDAO) CleanupHistory(ctx context.Context, before time.Time, maxAttempts int) error {
	for {
		result := d.db.WithContext(ctx).
			Where("status IN ? AND updated_at < ?", []string{JobDone, JobCancelled, JobSuppressed}, before).
			Limit(historyCleanupBatchSize).Delete(&NotificationJob{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected < historyCleanupBatchSize {
			break
		}
	}
	for {
		result := d.db.WithContext(ctx).
			Where("(status IN ? OR (status = ? AND attempts >= ?)) AND updated_at < ?",
				[]string{OutboxSent, OutboxSuppressed}, OutboxFailed, maxAttempts, before).
			Limit(historyCleanupBatchSize).Delete(&NotificationOutbox{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected < historyCleanupBatchSize {
			break
		}
	}
	return nil
}

func (d *ReminderDAO) Subscription(ctx context.Context, studentID string) (*LibraryReminderSubscription, error) {
	var sub LibraryReminderSubscription
	if err := d.db.WithContext(ctx).Where("student_id = ?", studentID).First(&sub).Error; err != nil {
		return nil, err
	}
	return &sub, nil
}

func (d *ReminderDAO) SubscriptionForUpdate(ctx context.Context, studentID string) (*LibraryReminderSubscription, error) {
	var sub LibraryReminderSubscription
	if err := d.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("student_id = ?", studentID).First(&sub).Error; err != nil {
		return nil, err
	}
	return &sub, nil
}

// SuppressAllWork 供顶层功能开关调用。保留记录而非删除可维持可审计性，
// 同时避免后续重新启用功能时投递过期任务。
func (d *ReminderDAO) SuppressAllWork(ctx context.Context) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&NotificationJob{}).Where("status IN ?", []string{JobPending, JobRunning}).Updates(map[string]any{
			"status": JobSuppressed, "version": gorm.Expr("version + 1"), "last_error": SuppressedReasonFeatureDisabled,
		}).Error; err != nil {
			return err
		}
		return tx.Model(&NotificationOutbox{}).Where("status IN ?", []string{OutboxPending, OutboxSending, OutboxFailed}).Updates(map[string]any{
			"status": OutboxSuppressed,
		}).Error
	})
}

func (d *ReminderDAO) MarkBaseline(ctx context.Context, studentID string, at time.Time, authStatus string) error {
	return d.db.WithContext(ctx).Model(&LibraryReminderSubscription{}).Where("student_id = ? AND enabled = ?", studentID, true).
		Updates(map[string]any{"baseline_completed": true, "last_full_refresh_at": at, "auth_status": authStatus}).Error
}

func (d *ReminderDAO) MarkRefreshFailure(ctx context.Context, studentID string, preferenceVersion int64, authStatus string) error {
	return d.db.WithContext(ctx).Model(&LibraryReminderSubscription{}).
		Where("student_id = ? AND preference_version = ?", studentID, preferenceVersion).
		Update("auth_status", authStatus).Error
}

func (d *ReminderDAO) MarkActiveScan(ctx context.Context, studentID string, at time.Time) error {
	return d.db.WithContext(ctx).Model(&LibraryReminderSubscription{}).Where("student_id = ?", studentID).Update("last_active_scan_at", at).Error
}

func (d *ReminderDAO) Reservation(ctx context.Context, studentID, reservationID string) (*ReservationSnapshot, error) {
	var row ReservationSnapshot
	if err := d.db.WithContext(ctx).Where("student_id = ? AND external_reservation_id = ?", studentID, reservationID).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (d *ReminderDAO) LatestReservationID(ctx context.Context, studentID string) (string, error) {
	var row ReservationSnapshot
	err := d.db.WithContext(ctx).Select("external_reservation_id").Where("student_id = ?", studentID).Order("last_seen_at DESC, id DESC").First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	return row.ExternalReservationID, err
}

func (d *ReminderDAO) SaveReservation(ctx context.Context, row *ReservationSnapshot) (bool, error) {
	var count int64
	if err := d.db.WithContext(ctx).Model(&ReservationSnapshot{}).Where("student_id = ? AND external_reservation_id = ?", row.StudentID, row.ExternalReservationID).Count(&count).Error; err != nil {
		return false, err
	}
	if row.FirstSeenAt.IsZero() {
		row.FirstSeenAt = row.LastSeenAt
	}
	err := d.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "student_id"}, {Name: "external_reservation_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"seat_id", "seat_label", "location", "make_date", "start_at", "end_at", "status", "raw_status", "last_seen_at", "updated_at"}),
	}).Create(row).Error
	return count == 0, err
}

func (d *ReminderDAO) SaveUserState(ctx context.Context, row *LibraryUserStateSnapshot) (*LibraryUserStateSnapshot, error) {
	var previous LibraryUserStateSnapshot
	err := d.db.WithContext(ctx).Where("student_id = ?", row.StudentID).First(&previous).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err := d.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "student_id"}}, DoUpdates: clause.AssignmentColumns([]string{"cycle_time_name", "breach_num", "score_num", "black_time", "black_message", "last_seen_at", "updated_at"})}).Create(row).Error; err != nil {
		return nil, err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &previous, nil
}

// AdvanceBlacklistEpisode 为一次新进入黑名单分配持久化版本，作为消息型事件的去重依据。
func (d *ReminderDAO) AdvanceBlacklistEpisode(ctx context.Context, studentID string, previous int) (int, error) {
	result := d.db.WithContext(ctx).Model(&LibraryUserStateSnapshot{}).
		Where("student_id = ? AND blacklist_episode = ?", studentID, previous).
		Update("blacklist_episode", previous+1)
	if result.Error != nil {
		return 0, result.Error
	}
	if result.RowsAffected != 1 {
		return 0, gorm.ErrRecordNotFound
	}
	return previous + 1, nil
}

func (d *ReminderDAO) UpsertJob(ctx context.Context, row *NotificationJob) error {
	if row.Status == "" {
		row.Status = JobPending
	}
	if row.Version == 0 {
		row.Version = 1
	}
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current NotificationJob
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("logical_key = ?", row.LogicalKey).First(&current).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Create(row).Error
		}
		if err != nil {
			return err
		}
		if current.Status == JobDone || current.Status == JobRunning {
			return nil
		}
		// 自动迁移后的旧任务没有不可变目标时间；刷新时就地补齐，不改变其领取时间。
		missingTarget := current.TargetAt.IsZero() && !row.TargetAt.IsZero()
		missingExpiry := current.ExpiresAt == nil && row.ExpiresAt != nil
		if current.Status == JobPending && (missingTarget || missingExpiry) {
			return tx.Model(&current).Updates(map[string]any{"target_at": row.TargetAt, "expires_at": row.ExpiresAt}).Error
		}
		// 已执行过的 pending 任务可能由 dispatchJob 重排到未来，不能被扫描任务拉回原始时间。
		if current.Status == JobPending && (current.Attempts > 0 || !row.RunAt.Before(current.RunAt)) {
			return nil
		}
		if current.Status == JobSuppressed && current.PreferenceVersion == row.PreferenceVersion && current.LastError != SuppressedReasonFeatureDisabled && current.LastError != SuppressedReasonNotificationTypeDisabled {
			return nil
		}
		return tx.Model(&current).Updates(map[string]any{"target_at": row.TargetAt, "expires_at": row.ExpiresAt, "run_at": row.RunAt, "status": JobPending, "preference_version": row.PreferenceVersion, "episode_version": row.EpisodeVersion, "version": gorm.Expr("version + 1"), "last_error": ""}).Error
	})
}

func (d *ReminderDAO) CancelReservationJobs(ctx context.Context, studentID, reservationID string) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&NotificationJob{}).
			Where("student_id = ? AND external_reservation_id = ? AND status IN ?", studentID, reservationID, []string{JobPending, JobRunning}).
			Updates(map[string]any{"status": JobCancelled, "version": gorm.Expr("version + 1")}).Error; err != nil {
			return err
		}
		return tx.Model(&NotificationOutbox{}).
			Where("student_id = ? AND external_reservation_id = ? AND status IN ?", studentID, reservationID, []string{OutboxPending, OutboxSending, OutboxFailed}).
			Update("status", OutboxSuppressed).Error
	})
}

func (d *ReminderDAO) CancelReservationJobTypes(ctx context.Context, studentID, reservationID string, types []string) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&NotificationJob{}).
			Where("student_id = ? AND external_reservation_id = ? AND type IN ? AND status IN ?", studentID, reservationID, types, []string{JobPending, JobRunning}).
			Updates(map[string]any{"status": JobCancelled, "version": gorm.Expr("version + 1")}).Error; err != nil {
			return err
		}
		return tx.Model(&NotificationOutbox{}).
			Where("student_id = ? AND external_reservation_id = ? AND type IN ? AND status IN ?", studentID, reservationID, types, []string{OutboxPending, OutboxSending, OutboxFailed}).
			Update("status", OutboxSuppressed).Error
	})
}

// CancelStaleReservationJobType 删除过期业务时间对应的任务，同时保留逻辑键仍与
// 当前预约时间匹配的任务和发件箱记录，从而保证预约改期安全，避免旧的 T-30/T-10
// 通知与新计划发生竞争。
func (d *ReminderDAO) CancelStaleReservationJobType(ctx context.Context, studentID, reservationID, notificationType, keepLogicalKey string) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		jobs := tx.Model(&NotificationJob{}).
			Where("student_id = ? AND external_reservation_id = ? AND type = ? AND status IN ?", studentID, reservationID, notificationType, []string{JobPending, JobRunning})
		if keepLogicalKey != "" {
			jobs = jobs.Where("logical_key <> ?", keepLogicalKey)
		}
		if err := jobs.Updates(map[string]any{"status": JobCancelled, "version": gorm.Expr("version + 1")}).Error; err != nil {
			return err
		}

		outbox := tx.Model(&NotificationOutbox{}).
			Where("student_id = ? AND external_reservation_id = ? AND type = ? AND status IN ?", studentID, reservationID, notificationType, []string{OutboxPending, OutboxSending, OutboxFailed})
		if keepLogicalKey != "" {
			outbox = outbox.Where("dedupe_key <> ?", keepLogicalKey)
		}
		return outbox.Update("status", OutboxSuppressed).Error
	})
}

func (d *ReminderDAO) EnqueueOutbox(ctx context.Context, row *NotificationOutbox) error {
	if row.Status == "" {
		row.Status = OutboxPending
	}
	if row.NextAttemptAt.IsZero() {
		row.NextAttemptAt = time.Now()
	}
	return d.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "dedupe_key"}}, DoNothing: true}).Create(row).Error
}

func (d *ReminderDAO) ClaimDueJobs(ctx context.Context, now time.Time, limit int) ([]NotificationJob, error) {
	currentSubscription := d.db.Model(&LibraryReminderSubscription{}).Select("1").
		Where("student_id = notification_jobs.student_id AND enabled = ? AND preference_version = notification_jobs.preference_version", true)
	if err := d.db.WithContext(ctx).Model(&NotificationJob{}).
		Where("status = ? AND NOT EXISTS (?)", JobPending, currentSubscription).
		Updates(map[string]any{"status": JobSuppressed, "last_error": "subscription changed"}).Error; err != nil {
		return nil, err
	}
	var candidates []NotificationJob
	if err := d.db.WithContext(ctx).Where("status = ? AND run_at <= ? AND EXISTS (?)", JobPending, now, currentSubscription).
		Order("run_at ASC, id ASC").Limit(limit).Find(&candidates).Error; err != nil {
		return nil, err
	}
	claimed := make([]NotificationJob, 0, len(candidates))
	for _, job := range candidates {
		result := d.db.WithContext(ctx).Model(&NotificationJob{}).
			Where("id = ? AND status = ? AND version = ? AND EXISTS (?)", job.ID, JobPending, job.Version,
				d.db.Model(&LibraryReminderSubscription{}).Select("1").Where("student_id = notification_jobs.student_id AND enabled = ? AND preference_version = notification_jobs.preference_version", true)).
			Updates(map[string]any{"status": JobRunning, "attempts": gorm.Expr("attempts + 1"), "version": gorm.Expr("version + 1")})
		if result.Error != nil {
			// 更新结果可能不明确，使用预期的新 claim 版本执行幂等释放。
			job.Status, job.Attempts, job.Version = JobRunning, job.Attempts+1, job.Version+1
			return append(claimed, job), result.Error
		}
		if result.RowsAffected == 1 {
			job.Status, job.Attempts, job.Version = JobRunning, job.Attempts+1, job.Version+1
			claimed = append(claimed, job)
		}
	}
	return claimed, nil
}

// ReleaseClaimedJobs 将本批尚未完成的任务重新置为待处理。
// 未实际执行完的 claim 不计入重试次数，后续调度可立即重新认领。
func (d *ReminderDAO) ReleaseClaimedJobs(ctx context.Context, jobs []NotificationJob) error {
	if len(jobs) == 0 {
		return nil
	}
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, job := range jobs {
			if err := tx.Model(&NotificationJob{}).
				Where("id = ? AND status = ? AND version = ?", job.ID, JobRunning, job.Version).
				Updates(map[string]any{
					"status":   JobPending,
					"attempts": gorm.Expr("CASE WHEN attempts > 0 THEN attempts - 1 ELSE 0 END"),
					"version":  gorm.Expr("version + 1"),
				}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (d *ReminderDAO) CompleteJobAndEnqueue(ctx context.Context, job NotificationJob, outbox *NotificationOutbox) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&NotificationJob{}).Where("id = ? AND status = ? AND version = ?", job.ID, JobRunning, job.Version).
			Update("status", JobDone)
		if result.Error != nil || result.RowsAffected != 1 {
			if result.Error != nil {
				return result.Error
			}
			return gorm.ErrRecordNotFound
		}
		return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "dedupe_key"}}, DoNothing: true}).Create(outbox).Error
	})
}

func (d *ReminderDAO) FinishJob(ctx context.Context, job NotificationJob, status, lastError string, next *time.Time) error {
	updates := map[string]any{"status": status, "last_error": lastError}
	if next != nil {
		updates["run_at"] = *next
		// 将本次上游复核收紧的有效期与重排时间一起持久化，沿用同一 claim 校验。
		updates["expires_at"] = job.ExpiresAt
	}
	result := d.db.WithContext(ctx).Model(&NotificationJob{}).
		Where("id = ? AND status = ? AND version = ?", job.ID, JobRunning, job.Version).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (d *ReminderDAO) ClaimOutbox(ctx context.Context, now time.Time, limit, maxAttempts int) ([]NotificationOutbox, error) {
	// 除 Feed 的最终校验外，偏好版本也作为本地安全校验。
	if err := d.db.WithContext(ctx).Model(&NotificationOutbox{}).Where("status IN ? AND NOT EXISTS (?)", []string{OutboxPending, OutboxFailed}, d.db.Model(&LibraryReminderSubscription{}).Select("1").Where("student_id = notification_outbox.student_id AND enabled = ? AND preference_version = notification_outbox.preference_version", true)).Update("status", OutboxSuppressed).Error; err != nil {
		return nil, err
	}
	var claimed []NotificationOutbox
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var candidates []NotificationOutbox
		if err := tx.Where("status IN ? AND next_attempt_at <= ? AND attempts < ?", []string{OutboxPending, OutboxFailed}, now, maxAttempts).Order("next_attempt_at ASC, id ASC").Limit(limit).Find(&candidates).Error; err != nil {
			return err
		}
		claimed = make([]NotificationOutbox, 0, len(candidates))
		for _, row := range candidates {
			result := tx.Model(&NotificationOutbox{}).Where("id = ? AND status IN ? AND attempts < ?", row.ID, []string{OutboxPending, OutboxFailed}, maxAttempts).Updates(map[string]any{"status": OutboxSending, "attempts": gorm.Expr("attempts + 1")})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 1 {
				row.Status, row.Attempts = OutboxSending, row.Attempts+1
				claimed = append(claimed, row)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return claimed, nil
}

// ReleaseClaimedOutbox 释放本批尚未开始投递的记录，并撤销 claim 预增的次数。
func (d *ReminderDAO) ReleaseClaimedOutbox(ctx context.Context, rows []NotificationOutbox) error {
	if len(rows) == 0 {
		return nil
	}
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, row := range rows {
			if err := tx.Model(&NotificationOutbox{}).
				Where("id = ? AND status = ? AND attempts = ?", row.ID, OutboxSending, row.Attempts).
				Updates(map[string]any{
					"status":   OutboxPending,
					"attempts": gorm.Expr("CASE WHEN attempts > 0 THEN attempts - 1 ELSE 0 END"),
				}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (d *ReminderDAO) FinishOutbox(ctx context.Context, row NotificationOutbox, status, lastError string, next *time.Time) error {
	updates := map[string]any{"status": status, "last_error": lastError}
	if next != nil {
		updates["next_attempt_at"] = *next
	}
	if status == OutboxSent {
		now := time.Now()
		updates["sent_at"] = now
	}
	result := d.db.WithContext(ctx).Model(&NotificationOutbox{}).
		Where("id = ? AND status = ? AND attempts = ?", row.ID, OutboxSending, row.Attempts).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (d *ReminderDAO) CanSendOutbox(ctx context.Context, row NotificationOutbox) (bool, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&NotificationOutbox{}).
		Where("id = ? AND status = ? AND attempts = ? AND EXISTS (?)", row.ID, OutboxSending, row.Attempts,
			d.db.Model(&LibraryReminderSubscription{}).Select("1").Where("student_id = notification_outbox.student_id AND enabled = ? AND preference_version = notification_outbox.preference_version", true)).
		Count(&count).Error
	return count == 1, err
}

func (d *ReminderDAO) LatestAwayEpisode(ctx context.Context, studentID, reservationID string) (*AwayEpisode, error) {
	var row AwayEpisode
	if err := d.db.WithContext(ctx).Where("student_id = ? AND external_reservation_id = ?", studentID, reservationID).Order("episode_version DESC").First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (d *ReminderDAO) LatestActiveAwayEpisode(ctx context.Context, studentID string) (*AwayEpisode, error) {
	var row AwayEpisode
	if err := d.db.WithContext(ctx).Where("student_id = ? AND state = ?", studentID, AwayStateAway).Order("updated_at DESC, id DESC").First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (d *ReminderDAO) SaveAwayEpisode(ctx context.Context, row *AwayEpisode) error {
	return d.db.WithContext(ctx).Save(row).Error
}
