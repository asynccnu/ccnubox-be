package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/asynccnu/ccnubox-be/be-library/conf"
	"github.com/asynccnu/ccnubox-be/be-library/crawler"
	"github.com/asynccnu/ccnubox-be/be-library/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-library/tool"
	feedv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feed/v1"
	userv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/user/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/asynccnu/ccnubox-be/common/pkg/metricsx"
	commontool "github.com/asynccnu/ccnubox-be/common/tool"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

const (
	NotificationReservationDiscovered = "RESERVATION_DISCOVERED"
	NotificationStart30               = "START_30"
	NotificationEnd10                 = "END_10"
	NotificationAway60                = "AWAY_60"
	NotificationAway80                = "AWAY_80"
	NotificationBreach                = "BREACH"
	NotificationBlacklisted           = "BLACKLISTED"

	reservationStatusMaxBytes   = 32
	notificationMessageMaxBytes = 8 << 10
	claimReleaseTimeout         = 5 * time.Second

	// 上游失败消息可达 4 MiB，批量样例只保留固定上限的截断文本，
	// 避免完整错误链随批量错误常驻内存并重复写入日志。
	subscriptionErrorSampleMaxBytes = 4 << 10
)

type notificationPayload struct {
	NotificationType string `json:"notification_type"`
	ReservationID    string `json:"reservation_id,omitempty"`
	SeatID           string `json:"seat_id,omitempty"`
	SeatLabel        string `json:"seat_label,omitempty"`
	Location         string `json:"location,omitempty"`
	StartAt          int64  `json:"start_at,omitempty"`
	EndAt            int64  `json:"end_at,omitempty"`
	TargetAt         int64  `json:"target_at,omitempty"`
	EpisodeVersion   int    `json:"episode_version,omitempty"`
	Message          string `json:"message,omitempty"`
}

type userTaskKey struct {
	studentID string
	taskType  string
}

type awayObservation struct {
	isAway         bool
	elapsedMinutes int
	elapsedKnown   bool
}

type userTaskState struct {
	preferenceVersion int64
	generation        int64
	running           bool
	nextAllowed       time.Time
}

type userTaskGate struct {
	mu     sync.Mutex
	states map[userTaskKey]userTaskState
}

// userTaskFinishFunc 用于结束一次已获得执行许可的用户任务。
// 无论任务成功或失败，都保留最小执行间隔，避免持续失败触发重试风暴。
type userTaskFinishFunc func(success bool, finishedAt time.Time)

func newUserTaskGate() *userTaskGate {
	return &userTaskGate{states: make(map[userTaskKey]userTaskState)}
}

// start 串行化同一用户的同类任务，并在任务结束后保留最小执行间隔。
// 偏好版本变化时立即使用新状态，旧任务的结束回调不会覆盖它。
func (g *userTaskGate) start(studentID, taskType string, preferenceVersion int64, now time.Time, interval time.Duration) (userTaskFinishFunc, bool) {
	key := userTaskKey{studentID: studentID, taskType: taskType}
	g.mu.Lock()
	state := g.states[key]
	if state.preferenceVersion == preferenceVersion && (state.running || now.Before(state.nextAllowed)) {
		g.mu.Unlock()
		return nil, false
	}
	state.preferenceVersion = preferenceVersion
	state.generation++
	state.running = true
	state.nextAllowed = now.Add(interval)
	g.states[key] = state
	generation := state.generation
	g.mu.Unlock()

	return func(success bool, finishedAt time.Time) {
		g.mu.Lock()
		defer g.mu.Unlock()
		current, ok := g.states[key]
		if !ok || current.generation != generation || current.preferenceVersion != preferenceVersion {
			return
		}
		if interval <= 0 {
			delete(g.states, key)
			return
		}
		current.running = false
		if !success {
			current.nextAllowed = finishedAt.Add(interval)
		}
		g.states[key] = current
	}, true
}

type ReminderService struct {
	dao            *dao.ReminderDAO
	crawler        crawler.ReminderCrawler
	user           userv1.UserServiceClient
	feed           FeedGateway
	config         conf.LibraryReminderConf
	logger         logger.Logger
	metrics        *metricsx.LibraryReminderMetrics
	now            func() time.Time
	preferenceLock *semaphore.Weighted
	userTaskGate   *userTaskGate
}

func NewReminderService(repo *dao.ReminderDAO, reminderCrawler crawler.ReminderCrawler, user userv1.UserServiceClient, feed FeedGateway, serverConf *conf.ServerConf, metricSet *metricsx.Metrics, l logger.Logger) *ReminderService {
	var reminderMetrics *metricsx.LibraryReminderMetrics
	if metricSet != nil {
		reminderMetrics = metricSet.Library
	}
	return &ReminderService{dao: repo, crawler: reminderCrawler, user: user, feed: feed, config: serverConf.Reminder(), logger: l, metrics: reminderMetrics, now: time.Now, preferenceLock: semaphore.NewWeighted(1), userTaskGate: newUserTaskGate()}
}

func (s *ReminderService) Enabled() bool { return s.config.Enabled }

func (s *ReminderService) RecoverOrphanedWork(ctx context.Context) error {
	return s.dao.RecoverOrphanedWork(ctx)
}

func (s *ReminderService) RecoverStaleWork(ctx context.Context) error {
	return s.dao.RecoverStaleWork(ctx, s.now().Add(-s.config.ClaimTimeout))
}

func (s *ReminderService) SuppressDisabledWork(ctx context.Context) error {
	return s.dao.SuppressAllWork(ctx)
}

func (s *ReminderService) SyncPreferences(ctx context.Context) (err error) {
	defer func() {
		if s.metrics == nil {
			return
		}
		result := "success"
		if err != nil {
			result = "error"
		}
		s.metrics.PreferenceSyncTotal.WithLabelValues(result).Inc()
	}()
	if err := s.preferenceLock.Acquire(ctx, 1); err != nil {
		return err
	}
	defer s.preferenceLock.Release(1)
	return s.syncPreferences(ctx)
}

func (s *ReminderService) syncPreferences(ctx context.Context) error {
	if !s.Enabled() {
		return nil
	}
	cursor, err := s.dao.Cursor(ctx)
	if err != nil {
		return err
	}
	if cursor < 0 {
		if err := s.rebuildPreferences(ctx); err != nil {
			return err
		}
		cursor, err = s.dao.Cursor(ctx)
		if err != nil {
			return err
		}
	}
	for {
		callCtx, cancel := s.remoteCallContext(ctx)
		changes, next, err := s.feed.PreferenceChanges(callCtx, cursor, 200)
		cancel()
		if err != nil {
			return err
		}
		if len(changes) == 0 {
			return s.refreshPendingBaselines(ctx)
		}
		daoChanges := make([]dao.PreferenceChange, 0, len(changes))
		lastRevision := cursor
		for _, change := range changes {
			if change.Revision <= lastRevision {
				return errors.New("feed returned an invalid library preference change")
			}
			lastRevision = change.Revision
			if !commontool.IsValidStudentID(change.StudentID) {
				// 历史脏标识不应阻塞全局游标；仍严格校验版本顺序，并留下可定位日志。
				s.logger.WithContext(ctx).Warn("skip invalid library preference student id", logger.Int64("revision", change.Revision))
				continue
			}
			daoChanges = append(daoChanges, dao.PreferenceChange{Revision: change.Revision, StudentID: change.StudentID, Enabled: change.Enabled})
			if s.metrics != nil && change.ChangedAt > 0 {
				lag := s.now().Unix() - change.ChangedAt
				if lag < 0 {
					lag = 0
				}
				s.metrics.PreferenceSyncLagSeconds.Set(float64(lag))
			}
		}
		if next != lastRevision {
			return errors.New("feed preference cursor does not match the last change")
		}
		enabled, err := s.dao.ApplyPreferenceChanges(ctx, daoChanges, next)
		if err != nil {
			return err
		}
		if s.config.ShouldBaselineOnEnable() {
			if err := s.forEachSubscription(ctx, enabled, s.RefreshUser); err != nil {
				return fmt.Errorf("enable baseline batch: %w", err)
			}
		}
		cursor = next
	}
}

func (s *ReminderService) rebuildPreferences(ctx context.Context) error {
	all, revision, err := s.loadReminderUsers(ctx)
	if err != nil {
		return err
	}
	enabled, err := s.dao.RebuildPreferences(ctx, all, revision)
	if err != nil {
		return err
	}
	if s.config.ShouldBaselineOnEnable() {
		if err := s.forEachSubscription(ctx, enabled, s.RefreshUser); err != nil {
			return fmt.Errorf("rebuild baseline batch: %w", err)
		}
	}
	return nil
}

func (s *ReminderService) refreshPendingBaselines(ctx context.Context) error {
	if !s.config.ShouldBaselineOnEnable() {
		return nil
	}
	pending, err := s.dao.PendingBaselineSubscriptions(ctx, 100000)
	if err != nil || len(pending) == 0 {
		return err
	}
	if err := s.forEachSubscription(ctx, pending, s.RefreshUser); err != nil {
		return fmt.Errorf("pending baseline batch: %w", err)
	}
	return nil
}

// 凌晨 3:15 全量校准 Feed 偏好
func (s *ReminderService) CalibratePreferences(ctx context.Context) error {
	if !s.Enabled() {
		return nil
	}
	if err := s.preferenceLock.Acquire(ctx, 1); err != nil {
		return err
	}
	defer s.preferenceLock.Release(1)
	all, _, err := s.loadReminderUsers(ctx)
	if err != nil {
		return err
	}
	enabled, err := s.dao.CalibratePreferences(ctx, all)
	if err != nil {
		return err
	}
	var baselineErr error
	if s.config.ShouldBaselineOnEnable() {
		if err := s.forEachSubscription(ctx, enabled, s.RefreshUser); err != nil {
			baselineErr = fmt.Errorf("calibrate baseline batch: %w", err)
		}
	}
	// 释放互斥锁前重放与分页全量查询并发的变更，消除全量快照与增量变更的顺序间隙。
	// baseline 批量失败也必须完成重放，否则快照加载/应用期间的偏好变更（如用户
	// 禁用提醒）会停留在过期状态，可能让 outbox 循环继续发送本应被抑制的通知。
	if err := s.syncPreferences(ctx); err != nil {
		return errors.Join(baselineErr, err)
	}
	return baselineErr
}

func (s *ReminderService) loadReminderUsers(ctx context.Context) ([]dao.PreferenceChange, int64, error) {
	after := int64(0)
	snapshotRevision := int64(0)
	firstPage := true
	var all []dao.PreferenceChange
	for {
		callCtx, cancel := s.remoteCallContext(ctx)
		users, next, pageSnapshotRevision, err := s.feed.ReminderUsers(callCtx, after, snapshotRevision, 200)
		cancel()
		if err != nil {
			return nil, 0, err
		}
		if pageSnapshotRevision < 0 || (!firstPage && pageSnapshotRevision != snapshotRevision) {
			return nil, 0, errors.New("feed reminder user snapshot revision changed during pagination")
		}
		snapshotRevision = pageSnapshotRevision
		firstPage = false
		for _, user := range users {
			if user.Revision <= 0 || user.Revision > snapshotRevision {
				return nil, 0, errors.New("feed returned an invalid library reminder user")
			}
			if !commontool.IsValidStudentID(user.StudentID) {
				s.logger.WithContext(ctx).Warn("skip invalid library reminder student id", logger.Int64("id", user.ID), logger.Int64("revision", user.Revision))
				continue
			}
			all = append(all, dao.PreferenceChange{Revision: user.Revision, StudentID: user.StudentID, Enabled: true})
		}
		if len(users) == 0 {
			break
		}
		if next <= after {
			return nil, 0, errors.New("feed reminder user cursor did not advance")
		}
		after = next
	}
	return all, snapshotRevision, nil
}

func (s *ReminderService) RefreshAll(ctx context.Context) error {
	if !s.Enabled() {
		return nil
	}
	rows, err := s.dao.EnabledSubscriptions(ctx, 100000)
	if err != nil {
		return err
	}
	return s.forEachSubscription(ctx, rows, s.RefreshUser)
}

func (s *ReminderService) RefreshUser(ctx context.Context, sub dao.LibraryReminderSubscription) (err error) {
	defer func() {
		if s.metrics == nil {
			return
		}
		result := "success"
		if err != nil {
			result = "error"
		}
		s.metrics.RefreshUsersTotal.WithLabelValues(result).Inc()
	}()
	return s.runUserOperation(ctx, sub, "full_refresh", s.config.FullRefreshMinInterval, s.refreshUserAttempt)
}

func (s *ReminderService) refreshUserAttempt(ctx context.Context, sub dao.LibraryReminderSubscription) error {
	current, err := s.dao.Subscription(ctx, sub.StudentID)
	if err != nil || !current.Enabled || current.PreferenceVersion != sub.PreferenceVersion {
		if err != nil {
			return err
		}
		return nil
	}
	token, err := s.libraryToken(ctx, sub.StudentID)
	if err != nil {
		_ = s.dao.MarkRefreshFailure(ctx, sub.StudentID, sub.PreferenceVersion, libraryTokenFailureStatus(err))
		return stageFailure(ctx, err, "get_library_token", "token_rpc")
	}
	if current, err := s.subscriptionStillCurrent(ctx, sub); err != nil || !current {
		return err
	}
	today, err := s.crawler.GetTodayReservations(ctx, token)
	if err != nil {
		_ = s.dao.MarkRefreshFailure(ctx, sub.StudentID, sub.PreferenceVersion, dao.SubscriptionAuthUpstreamUnknown)
		return stageFailure(ctx, err, "get_today_reservations", upstreamFailureKind(err))
	}
	if current, err := s.subscriptionStillCurrent(ctx, sub); err != nil || !current {
		return err
	}
	watermark, err := s.dao.LatestReservationID(ctx, sub.StudentID)
	if err != nil {
		return err
	}
	history, err := s.crawler.GetRecentHistory(ctx, token, crawler.HistoryWatermark{ReservationID: watermark})
	if err != nil {
		_ = s.dao.MarkRefreshFailure(ctx, sub.StudentID, sub.PreferenceVersion, dao.SubscriptionAuthUpstreamUnknown)
		return stageFailure(ctx, err, "get_recent_history", upstreamFailureKind(err))
	}
	if current, err := s.subscriptionStillCurrent(ctx, sub); err != nil || !current {
		return err
	}
	state, err := s.crawler.GetUserState(ctx, token)
	if err != nil {
		_ = s.dao.MarkRefreshFailure(ctx, sub.StudentID, sub.PreferenceVersion, dao.SubscriptionAuthUpstreamUnknown)
		return stageFailure(ctx, err, "get_user_state", upstreamFailureKind(err))
	}
	// 部分历史分页可安全用于更新插入和新增事实，但不能据此执行“缺失即取消”的状态转换。
	reservations := make(map[string]crawler.ReminderReservation, len(today)+len(history.Reservations))
	for _, row := range history.Reservations {
		reservations[row.ID] = row
	}
	for _, row := range today {
		reservations[row.ID] = row
	}
	now := s.now()
	if err := s.dao.Transaction(ctx, func(txDAO *dao.ReminderDAO) error {
		locked, err := txDAO.SubscriptionForUpdate(ctx, sub.StudentID)
		if err != nil {
			return err
		}
		if !locked.Enabled || locked.PreferenceVersion != sub.PreferenceVersion {
			return nil
		}
		txService := *s
		txService.dao = txDAO
		for _, row := range reservations {
			if err := txService.reconcileReservation(ctx, *locked, row, now); err != nil {
				return err
			}
		}
		if err := txService.reconcileUserState(ctx, *locked, state, now); err != nil {
			return err
		}
		return txDAO.MarkBaseline(ctx, sub.StudentID, now, dao.SubscriptionAuthOK)
	}); err != nil {
		return stageFailure(ctx, err, "save_reservation_state", "database")
	}
	return nil
}

func (s *ReminderService) reconcileReservation(ctx context.Context, sub dao.LibraryReminderSubscription, row crawler.ReminderReservation, now time.Time) error {
	start, end, err := row.Times()
	if err != nil {
		return err
	}
	status := strings.ToUpper(strings.TrimSpace(row.Status))
	if status != "" && !terminalReservationStatus(status) && status != "USING" && status != "WAIT" && status != "SUCCESS" {
		s.logger.WithContext(ctx).Warn("unknown library reservation status", logger.String("student_id", maskStudentID(sub.StudentID)), logger.String("status", status))
		if s.metrics != nil {
			s.metrics.UnknownReservationStatusTotal.Inc()
		}
	}
	storedStatus := truncateUTF8(status, reservationStatusMaxBytes)
	snapshot := &dao.ReservationSnapshot{StudentID: sub.StudentID, ExternalReservationID: row.ID, SeatID: row.SeatID, SeatLabel: row.SeatLabel, Location: row.Location, MakeDate: row.MakeDateStr, StartAt: start, EndAt: end, Status: storedStatus, RawStatus: row.Status, FirstSeenAt: now, LastSeenAt: now}
	created, err := s.dao.SaveReservation(ctx, snapshot)
	if err != nil {
		return err
	}
	if terminalReservationStatus(status) || !end.After(now) {
		return s.dao.CancelReservationJobs(ctx, sub.StudentID, row.ID)
	}
	if shouldEnqueueReservationDiscovered(created, sub.BaselineCompleted, end, now, s.config.NotificationTypes.ReservationDiscovered) {
		if err := s.enqueueFact(ctx, sub, NotificationReservationDiscovered, row.ID, 0, start, end, now); err != nil {
			return err
		}
	}
	if s.config.NotificationTypes.Start30 && start.After(now) {
		targetAt := start.Add(-30 * time.Minute)
		runAt := targetAt
		if runAt.Before(now) {
			runAt = now
		}
		key := reminderJobKey(sub.StudentID, row.ID, NotificationStart30, start.Unix())
		if err := s.dao.CancelStaleReservationJobType(ctx, sub.StudentID, row.ID, NotificationStart30, key); err != nil {
			return err
		}
		if err := s.scheduleJob(ctx, sub, NotificationStart30, row.ID, 0, runAt, targetAt, &start, start.Unix()); err != nil {
			return err
		}
	} else if err := s.dao.CancelReservationJobTypes(ctx, sub.StudentID, row.ID, []string{NotificationStart30}); err != nil {
		return err
	}
	if s.config.NotificationTypes.End10 && end.After(now) {
		targetAt := end.Add(-10 * time.Minute)
		runAt := targetAt
		if runAt.Before(now) {
			runAt = now
		}
		key := reminderJobKey(sub.StudentID, row.ID, NotificationEnd10, end.Unix())
		if err := s.dao.CancelStaleReservationJobType(ctx, sub.StudentID, row.ID, NotificationEnd10, key); err != nil {
			return err
		}
		if err := s.scheduleJob(ctx, sub, NotificationEnd10, row.ID, 0, runAt, targetAt, &end, end.Unix()); err != nil {
			return err
		}
	} else if err := s.dao.CancelReservationJobTypes(ctx, sub.StudentID, row.ID, []string{NotificationEnd10}); err != nil {
		return err
	}
	return nil
}

func shouldEnqueueReservationDiscovered(created, baselineCompleted bool, end, now time.Time, enabled bool) bool {
	return enabled && created && baselineCompleted && end.After(now)
}

func (s *ReminderService) reconcileUserState(ctx context.Context, sub dao.LibraryReminderSubscription, state crawler.LibraryUserState, now time.Time) error {
	row := &dao.LibraryUserStateSnapshot{StudentID: sub.StudentID, CycleTimeName: state.CycleTimeName, BreachNum: state.BreachNum, ScoreNum: state.ScoreNum, BlackTime: state.BlackTime, BlackMessage: state.BlackMessage, LastSeenAt: now}
	previous, err := s.dao.SaveUserState(ctx, row)
	if err != nil || previous == nil || !sub.BaselineCompleted {
		return err
	}
	if s.config.NotificationTypes.Breach && state.CycleTimeName == previous.CycleTimeName && state.BreachNum > previous.BreachNum {
		key := boundedDedupeKey(fmt.Sprintf("%s:%s:%d:%s", sub.StudentID, state.CycleTimeName, state.BreachNum, NotificationBreach))
		payload := notificationPayload{NotificationType: NotificationBreach, TargetAt: now.Unix()}
		if err := s.enqueuePayload(ctx, sub, key, NotificationBreach, payload); err != nil {
			return err
		}
	}
	previousBlackTime := strings.TrimSpace(previous.BlackTime)
	blackTime := strings.TrimSpace(state.BlackTime)
	wasBlacklisted := previousBlackTime != "" || strings.TrimSpace(previous.BlackMessage) != ""
	isBlacklisted := blackTime != "" || strings.TrimSpace(state.BlackMessage) != ""
	blackTimeChanged := previousBlackTime != "" && blackTime != "" && blackTime != previousBlackTime
	enteredBlacklist := (!wasBlacklisted && isBlacklisted) || blackTimeChanged
	if s.config.NotificationTypes.Blacklisted && enteredBlacklist {
		episode, err := s.dao.AdvanceBlacklistEpisode(ctx, sub.StudentID, previous.BlacklistEpisode)
		if err != nil {
			return err
		}
		key := boundedDedupeKey(fmt.Sprintf("%s:%s:%d", sub.StudentID, NotificationBlacklisted, episode))
		message := strings.TrimSpace(state.BlackMessage)
		if message == "" && blackTime != "" {
			message = "图书馆预约权限已暂停至 " + blackTime + "。"
		}
		payload := notificationPayload{NotificationType: NotificationBlacklisted, TargetAt: now.Unix(), EpisodeVersion: episode, Message: message}
		return s.enqueuePayload(ctx, sub, key, NotificationBlacklisted, payload)
	}
	return nil
}

// 仅高频率扫描正在使用中的座位
// 减少性能开销
func (s *ReminderService) ScanActive(ctx context.Context) error {
	if !s.Enabled() {
		return nil
	}
	rows, err := s.dao.ActiveSubscriptions(ctx, s.now(), 100000)
	if err != nil {
		return err
	}
	if s.metrics != nil {
		s.metrics.ActiveReservations.Set(float64(len(rows)))
	}
	return s.forEachSubscription(ctx, rows, s.scanActiveUser)
}

func (s *ReminderService) scanActiveUser(ctx context.Context, sub dao.LibraryReminderSubscription) error {
	return s.runUserOperation(ctx, sub, "active_scan", s.config.ActiveScanMinInterval, s.scanActiveUserAttempt)
}

func (s *ReminderService) scanActiveUserAttempt(ctx context.Context, sub dao.LibraryReminderSubscription) error {
	currentSub, err := s.dao.Subscription(ctx, sub.StudentID)
	if err != nil || !currentSub.Enabled || currentSub.PreferenceVersion != sub.PreferenceVersion {
		return err
	}
	token, err := s.libraryToken(ctx, sub.StudentID)
	if err != nil {
		return stageFailure(ctx, err, "get_library_token", "token_rpc")
	}
	if current, err := s.subscriptionStillCurrent(ctx, sub); err != nil || !current {
		return err
	}
	current, err := s.crawler.GetCurrentReservation(ctx, token)
	if err != nil {
		return stageFailure(ctx, err, "get_current_reservation", upstreamFailureKind(err))
	}
	now := s.now()
	if err := s.dao.Transaction(ctx, func(txDAO *dao.ReminderDAO) error {
		locked, err := txDAO.SubscriptionForUpdate(ctx, sub.StudentID)
		if err != nil {
			return err
		}
		if !locked.Enabled || locked.PreferenceVersion != sub.PreferenceVersion {
			return nil
		}
		txService := *s
		txService.dao = txDAO
		return txService.applyActiveObservation(ctx, *locked, current, now)
	}); err != nil {
		return stageFailure(ctx, err, "save_active_observation", "database")
	}
	return nil
}

func (s *ReminderService) applyActiveObservation(ctx context.Context, sub dao.LibraryReminderSubscription, current *crawler.ReminderReservation, now time.Time) error {
	var end time.Time
	if current != nil {
		var err error
		_, end, err = current.Times()
		if err != nil {
			return err
		}
		if terminalReservationStatus(current.Status) || !end.After(now) {
			// 先保存终止/过期快照并取消关联工作，避免继续扫描或按旧快照发送提醒。
			if err := s.reconcileReservation(ctx, sub, *current, now); err != nil {
				return err
			}
			current = nil
		}
	}
	if current == nil {
		if episode, findErr := s.dao.LatestActiveAwayEpisode(ctx, sub.StudentID); findErr == nil {
			episode.State = dao.AwayStateEnded
			if err := s.dao.SaveAwayEpisode(ctx, episode); err != nil {
				return err
			}
			if err := s.dao.CancelReservationJobTypes(ctx, sub.StudentID, episode.ExternalReservationID, []string{NotificationAway60, NotificationAway80}); err != nil {
				return err
			}
		} else if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}
		return s.dao.MarkActiveScan(ctx, sub.StudentID, now)
	}
	if oldEpisode, findErr := s.dao.LatestActiveAwayEpisode(ctx, sub.StudentID); findErr == nil && oldEpisode.ExternalReservationID != current.ID {
		oldEpisode.State = dao.AwayStateEnded
		if err := s.dao.SaveAwayEpisode(ctx, oldEpisode); err != nil {
			return err
		}
		if err := s.dao.CancelReservationJobTypes(ctx, sub.StudentID, oldEpisode.ExternalReservationID, []string{NotificationAway60, NotificationAway80, NotificationEnd10}); err != nil {
			return err
		}
	} else if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
		return findErr
	}
	if _, err := s.dao.Reservation(ctx, sub.StudentID, current.ID); errors.Is(err, gorm.ErrRecordNotFound) {
		if recErr := s.reconcileReservation(ctx, sub, *current, now); recErr != nil {
			return recErr
		}
	} else if err != nil {
		return err
	}
	away := currentAwayObservation(*current, now)
	episode, err := s.dao.LatestAwayEpisode(ctx, sub.StudentID, current.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if !away.isAway {
		if err == nil && episode.State == dao.AwayStateAway {
			episode.State = dao.AwayStateReturned
			if err := s.dao.SaveAwayEpisode(ctx, episode); err != nil {
				return err
			}
			if err := s.dao.CancelReservationJobTypes(ctx, sub.StudentID, current.ID, []string{NotificationAway60, NotificationAway80}); err != nil {
				return err
			}
		}
		return s.dao.MarkActiveScan(ctx, sub.StudentID, now)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		episode = nil
	}
	episode = nextAwayEpisode(episode, sub.StudentID, current.ID, away.elapsedMinutes, now)
	version := episode.EpisodeVersion
	if err := s.dao.SaveAwayEpisode(ctx, episode); err != nil {
		return err
	}
	if s.config.NotificationTypes.Away60 {
		targetAt := episode.AwayStartedAt.Add(60 * time.Minute)
		if err := s.scheduleJob(ctx, sub, NotificationAway60, current.ID, version, targetAt, targetAt, &end, int64(version)); err != nil {
			return err
		}
	}
	if s.config.NotificationTypes.Away80 {
		targetAt := episode.AwayStartedAt.Add(80 * time.Minute)
		if err := s.scheduleJob(ctx, sub, NotificationAway80, current.ID, version, targetAt, targetAt, &end, int64(version)); err != nil {
			return err
		}
	}
	return s.dao.MarkActiveScan(ctx, sub.StudentID, now)
}

// 只结束本次观察对应的 episode，避免旧任务覆盖同一预约的新一轮暂离。
// 与扫描共用订阅行锁，episode 状态和关联任务、outbox 必须一起提交。
func (s *ReminderService) endAwayEpisode(ctx context.Context, studentID, reservationID string, preferenceVersion int64, episodeVersion int, state string) (bool, error) {
	ended := false
	err := s.dao.Transaction(ctx, func(txDAO *dao.ReminderDAO) error {
		sub, err := txDAO.SubscriptionForUpdate(ctx, studentID)
		if err != nil {
			return err
		}
		if !sub.Enabled || sub.PreferenceVersion != preferenceVersion {
			return nil
		}
		episode, err := txDAO.LatestAwayEpisode(ctx, studentID, reservationID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if episode.State != dao.AwayStateAway || episode.EpisodeVersion != episodeVersion {
			return nil
		}
		episode.State = state
		if err := txDAO.SaveAwayEpisode(ctx, episode); err != nil {
			return err
		}
		if err := txDAO.CancelReservationJobTypes(ctx, studentID, reservationID, []string{NotificationAway60, NotificationAway80}); err != nil {
			return err
		}
		ended = true
		return nil
	})
	return ended, err
}

func (s *ReminderService) awayEpisodeCurrent(ctx context.Context, studentID, reservationID string, version int) (bool, error) {
	// 迁移策略：无有效版本或缺少 episode 的历史暂离任务不补发，等待扫描建立可信状态。
	if version <= 0 {
		return false, nil
	}
	episode, err := s.dao.LatestAwayEpisode(ctx, studentID, reservationID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return episode.State == dao.AwayStateAway && episode.EpisodeVersion == version, nil
}

func nextAwayEpisode(existing *dao.AwayEpisode, studentID, reservationID string, awayMinutes int, observedAt time.Time) *dao.AwayEpisode {
	if existing != nil && existing.State == dao.AwayStateAway {
		existing.LastAwayMinutes = awayMinutes
		return existing
	}
	version := 1
	if existing != nil {
		version = existing.EpisodeVersion + 1
	}
	return &dao.AwayEpisode{
		StudentID:             studentID,
		ExternalReservationID: reservationID,
		EpisodeVersion:        version,
		AwayStartedAt:         observedAt.Add(-time.Duration(awayMinutes) * time.Minute),
		LastAwayMinutes:       awayMinutes,
		State:                 dao.AwayStateAway,
	}
}

func currentAwayObservation(row crawler.ReminderReservation, observedAt time.Time) awayObservation {
	if terminalReservationStatus(row.Status) {
		return awayObservation{}
	}
	if row.AwayTimeM > 0 {
		return awayObservation{isAway: true, elapsedMinutes: row.AwayTimeM, elapsedKnown: true}
	}
	if strings.TrimSpace(row.AwayRange) == "" {
		return awayObservation{}
	}
	minutes, ok := awayMinutesFromRange(row.AwayRange, row.MakeDateStr, observedAt)
	if ok {
		return awayObservation{isAway: true, elapsedMinutes: minutes, elapsedKnown: true}
	}
	return awayObservation{isAway: true}
}

var awayRangeClockRE = regexp.MustCompile(`(\d{1,2}):(\d{2})`)

func awayMinutesFromRange(awayRange, makeDate string, observedAt time.Time) (int, bool) {
	match := awayRangeClockRE.FindStringSubmatch(awayRange)
	if match == nil {
		return 0, false
	}
	hour, err := strconv.Atoi(match[1])
	if err != nil || hour < 0 || hour > 23 {
		return 0, false
	}
	minute, err := strconv.Atoi(match[2])
	if err != nil || minute < 0 || minute > 59 {
		return 0, false
	}
	loc := tool.GetLocation()
	observed := observedAt.In(loc)
	day := time.Date(observed.Year(), observed.Month(), observed.Day(), 0, 0, 0, 0, loc)
	if strings.TrimSpace(makeDate) != "" {
		parsed, err := time.ParseInLocation("2006-01-02", makeDate, loc)
		if err != nil {
			return 0, false
		}
		day = parsed
	}
	startedAt := day.Add(time.Duration(hour)*time.Hour + time.Duration(minute)*time.Minute)
	if startedAt.After(observed) && startedAt.Sub(observed) > 12*time.Hour {
		startedAt = startedAt.Add(-24 * time.Hour)
	}
	if startedAt.After(observed) {
		return 0, false
	}
	return int(observed.Sub(startedAt) / time.Minute), true
}

func (s *ReminderService) DispatchJobs(ctx context.Context) (err error) {
	defer s.observeWorkMetrics(ctx)
	if !s.Enabled() {
		return nil
	}
	processed := 0
	for processed < s.config.JobDispatchBudget {
		limit := min(s.config.JobDispatchBatchSize, s.config.JobDispatchBudget-processed)
		jobs, claimErr := s.dao.ClaimDueJobs(ctx, s.now(), limit)
		if claimErr != nil {
			// claim 逐条执行，失败前已认领的任务也必须使用独立上下文释放。
			s.releaseClaimedJobs(ctx, jobs)
			return claimErr
		}
		if len(jobs) == 0 {
			return nil
		}
		processed += len(jobs)
		if err := s.dispatchJobBatch(ctx, jobs); err != nil {
			return err
		}
		if len(jobs) < limit {
			return nil
		}
	}
	return nil
}

func (s *ReminderService) dispatchJobBatch(ctx context.Context, jobs []dao.NotificationJob) error {
	limit := min(s.config.JobDispatchConcurrency, len(jobs))
	sem := make(chan struct{}, limit)
	var wg sync.WaitGroup
	var launchErr error
launch:
	for i := range jobs {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			s.releaseClaimedJobs(ctx, jobs[i:])
			launchErr = ctx.Err()
			break launch
		}
		job := jobs[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			if err := s.dispatchJob(ctx, job); err != nil {
				s.logger.WithContext(ctx).Warn("library reminder job failed", logger.Int64("job_id", job.ID), logger.String("notification_type", job.Type), logger.Error(err))
				// dispatchJob 可能已完成重排；释放操作只会命中仍属于当前 claim 的 running 记录。
				s.releaseClaimedJobs(ctx, []dao.NotificationJob{job})
			}
		}()
	}
	wg.Wait()
	if launchErr != nil {
		return launchErr
	}
	return ctx.Err()
}

func (s *ReminderService) releaseClaimedJobs(parent context.Context, jobs []dao.NotificationJob) {
	if len(jobs) == 0 {
		return
	}
	releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(parent), claimReleaseTimeout)
	defer cancel()
	if err := s.dao.ReleaseClaimedJobs(releaseCtx, jobs); err != nil {
		s.logger.WithContext(parent).Warn("release claimed library reminder jobs failed", logger.Int("count", len(jobs)), logger.Error(err))
	}
}

func (s *ReminderService) dispatchJob(ctx context.Context, job dao.NotificationJob) error {
	if !s.notificationEnabled(job.Type) {
		return s.dao.FinishJob(ctx, job, dao.JobSuppressed, dao.SuppressedReasonNotificationTypeDisabled, nil)
	}
	if job.ExpiresAt != nil && !job.ExpiresAt.After(s.now()) {
		return s.dao.FinishJob(ctx, job, dao.JobSuppressed, "notification expired", nil)
	}
	sub, err := s.dao.Subscription(ctx, job.StudentID)
	if err != nil {
		return s.retryJob(ctx, job, err)
	}
	if !sub.Enabled || sub.PreferenceVersion != job.PreferenceVersion {
		return s.dao.FinishJob(ctx, job, dao.JobSuppressed, "subscription changed", nil)
	}
	var verifiedReservation *crawler.ReminderReservation
	if job.Type == NotificationAway60 || job.Type == NotificationAway80 {
		if current, err := s.awayEpisodeCurrent(ctx, job.StudentID, job.ExternalReservationID, job.EpisodeVersion); err != nil {
			return s.retryJob(ctx, job, err)
		} else if !current {
			return s.dao.FinishJob(ctx, job, dao.JobSuppressed, "away episode changed", nil)
		}
		threshold := 60
		if job.Type == NotificationAway80 {
			threshold = 80
		}
		token, err := s.libraryToken(ctx, job.StudentID)
		if err != nil {
			return s.retryJob(ctx, job, err)
		}
		// 查询失败仅表示订阅状态未知，不能当作用户关闭提醒。
		if current, err := s.subscriptionStillCurrent(ctx, *sub); err != nil {
			return s.retryJob(ctx, job, err)
		} else if !current {
			return s.dao.FinishJob(ctx, job, dao.JobSuppressed, "subscription changed", nil)
		}
		current, err := s.crawler.GetCurrentReservation(ctx, token)
		if err != nil {
			return s.retryJob(ctx, job, err)
		}
		now := s.now()
		away := awayObservation{}
		state := dao.AwayStateEnded
		var end time.Time
		if current != nil && current.ID == job.ExternalReservationID && !terminalReservationStatus(current.Status) {
			_, end, err = current.Times()
			if err != nil {
				return s.retryJob(ctx, job, err)
			}
			if end.After(now) {
				away = currentAwayObservation(*current, now)
				state = dao.AwayStateReturned
			}
		}
		if !away.isAway {
			ended, err := s.endAwayEpisode(ctx, job.StudentID, job.ExternalReservationID, job.PreferenceVersion, job.EpisodeVersion, state)
			if err != nil {
				return s.retryJob(ctx, job, err)
			}
			if ended {
				// 结束 episode 已原子取消当前 claim，无需再次 FinishJob。
				return nil
			}
			return s.dao.FinishJob(ctx, job, dao.JobSuppressed, "away condition no longer holds", nil)
		}
		// 为旧的无有效期任务补齐边界，且不能随预约延长而延长旧提醒的有效期。
		if job.ExpiresAt == nil || end.Before(*job.ExpiresAt) {
			job.ExpiresAt = &end
		}
		if away.elapsedKnown && away.elapsedMinutes < threshold {
			next := now.Add(time.Duration(threshold-away.elapsedMinutes) * time.Minute)
			if !next.Before(*job.ExpiresAt) {
				return s.dao.FinishJob(ctx, job, dao.JobSuppressed, "notification expired before retry", nil)
			}
			return s.dao.FinishJob(ctx, job, dao.JobPending, "", &next)
		}
		verifiedReservation = current
	}
	if job.Type == NotificationStart30 || job.Type == NotificationEnd10 {
		token, err := s.libraryToken(ctx, job.StudentID)
		if err != nil {
			return s.retryJob(ctx, job, err)
		}
		if current, err := s.subscriptionStillCurrent(ctx, *sub); err != nil {
			return s.retryJob(ctx, job, err)
		} else if !current {
			return s.dao.FinishJob(ctx, job, dao.JobSuppressed, "subscription changed", nil)
		}
		reservations, err := s.crawler.GetTodayReservations(ctx, token)
		if err != nil {
			return s.retryJob(ctx, job, err)
		}
		for i := range reservations {
			reservation := &reservations[i]
			if reservation.ID != job.ExternalReservationID || terminalReservationStatus(reservation.Status) {
				continue
			}
			start, end, err := reservation.Times()
			if err != nil {
				return s.retryJob(ctx, job, err)
			}
			expectedTarget, expectedExpiry := start.Add(-30*time.Minute), start
			if job.Type == NotificationEnd10 {
				expectedTarget, expectedExpiry = end.Add(-10*time.Minute), end
			}
			if job.TargetAt.IsZero() || !job.TargetAt.Equal(expectedTarget) || job.ExpiresAt == nil || !job.ExpiresAt.Equal(expectedExpiry) {
				return s.dao.FinishJob(ctx, job, dao.JobSuppressed, "reservation time changed", nil)
			}
			if !expectedExpiry.After(s.now()) {
				return s.dao.FinishJob(ctx, job, dao.JobSuppressed, "notification expired", nil)
			}
			verifiedReservation = reservation
			break
		}
		if verifiedReservation == nil {
			return s.dao.FinishJob(ctx, job, dao.JobSuppressed, "reservation no longer active", nil)
		}
	}
	var start, end time.Time
	var seatID, seatLabel, location string
	if verifiedReservation != nil {
		start, end, _ = verifiedReservation.Times()
		seatID, seatLabel, location = verifiedReservation.SeatID, verifiedReservation.SeatLabel, verifiedReservation.Location
	} else if job.ExternalReservationID != "" {
		reservation, findErr := s.dao.Reservation(ctx, job.StudentID, job.ExternalReservationID)
		if findErr != nil {
			return s.retryJob(ctx, job, findErr)
		}
		start, end, seatID, seatLabel, location = reservation.StartAt, reservation.EndAt, reservation.SeatID, reservation.SeatLabel, reservation.Location
	}
	targetAt := job.TargetAt
	if targetAt.IsZero() {
		// 兼容自动迁移后尚未来得及由扫描补齐目标时间的暂离任务。
		targetAt = job.RunAt
	}
	payload := notificationPayload{NotificationType: job.Type, ReservationID: job.ExternalReservationID, SeatID: seatID, SeatLabel: seatLabel, Location: location, StartAt: start.Unix(), EndAt: end.Unix(), TargetAt: targetAt.Unix(), EpisodeVersion: job.EpisodeVersion}
	raw, err := json.Marshal(payload)
	if err != nil {
		return s.retryJob(ctx, job, err)
	}
	outbox := &dao.NotificationOutbox{DedupeKey: job.LogicalKey, StudentID: job.StudentID, ExternalReservationID: job.ExternalReservationID, PreferenceVersion: job.PreferenceVersion, Type: job.Type, Payload: raw, Status: dao.OutboxPending, NextAttemptAt: s.now(), ExpiresAt: job.ExpiresAt}
	return s.dao.CompleteJobAndEnqueue(ctx, job, outbox)
}

func (s *ReminderService) retryJob(ctx context.Context, job dao.NotificationJob, cause error) error {
	if job.Attempts >= s.config.RetryMaxAttempts {
		return s.dao.FinishJob(ctx, job, dao.JobSuppressed, "upstream state remained unknown", nil)
	}
	delay := time.Duration(1<<min(job.Attempts, 6)) * time.Minute
	next := s.now().Add(delay)
	if job.ExpiresAt != nil && !next.Before(*job.ExpiresAt) {
		return s.dao.FinishJob(ctx, job, dao.JobSuppressed, "notification expired before retry", nil)
	}
	if err := s.dao.FinishJob(ctx, job, dao.JobPending, "transient failure", &next); err != nil {
		return err
	}
	return cause
}

// CleanupHistory 清理超过保留期的终态任务与 outbox 记录，控制历史表规模。
func (s *ReminderService) CleanupHistory(ctx context.Context) error {
	if !s.Enabled() {
		return nil
	}
	before := s.now().AddDate(0, 0, -s.config.HistoryRetentionDays)
	return s.dao.CleanupHistory(ctx, before, s.config.RetryMaxAttempts)
}

func (s *ReminderService) SendOutbox(ctx context.Context) (err error) {
	defer s.observeWorkMetrics(ctx)
	if !s.Enabled() {
		return nil
	}
	rows, err := s.dao.ClaimOutbox(ctx, s.now(), 100, s.config.RetryMaxAttempts)
	if err != nil {
		return err
	}
	for i := range rows {
		if err := ctx.Err(); err != nil {
			s.releaseClaimedOutbox(ctx, rows[i:])
			return err
		}
		row := rows[i]
		if err := s.sendOutboxRow(ctx, row); err != nil {
			s.logger.WithContext(ctx).Warn("library reminder outbox row failed",
				logger.Int64("outbox_id", row.ID),
				logger.String("notification_type", row.Type),
				logger.String("student_id", maskStudentID(row.StudentID)),
				logger.Int("attempt", row.Attempts),
				logger.Error(err),
			)
			s.releaseOutboxAfterFailure(ctx, row)
		}
	}
	return nil
}

func (s *ReminderService) sendOutboxRow(ctx context.Context, row dao.NotificationOutbox) error {
	if !s.notificationEnabled(row.Type) {
		return s.dao.FinishOutbox(ctx, row, dao.OutboxSuppressed, dao.SuppressedReasonNotificationTypeDisabled, nil)
	}
	if row.ExpiresAt != nil && !row.ExpiresAt.After(s.now()) {
		return s.dao.FinishOutbox(ctx, row, dao.OutboxSuppressed, "notification expired", nil)
	}
	if s.config.IsDryRun() {
		return s.dao.FinishOutbox(ctx, row, dao.OutboxSuppressed, "dry_run", nil)
	}
	var payload notificationPayload
	if err := json.Unmarshal(row.Payload, &payload); err != nil {
		return s.dao.FinishOutbox(ctx, row, dao.OutboxSuppressed, "invalid payload", nil)
	}
	canSend, err := s.dao.CanSendOutbox(ctx, row)
	if err != nil {
		return err
	}
	if !canSend {
		return s.dao.FinishOutbox(ctx, row, dao.OutboxSuppressed, "subscription changed", nil)
	}
	if row.Type == NotificationStart30 || row.Type == NotificationEnd10 {
		reservation, err := s.dao.Reservation(ctx, row.StudentID, row.ExternalReservationID)
		if err != nil {
			return err
		}
		if terminalReservationStatus(reservation.Status) {
			return s.dao.FinishOutbox(ctx, row, dao.OutboxSuppressed, "reservation no longer active", nil)
		}
		expectedTarget, expectedExpiry := reservation.StartAt.Add(-30*time.Minute), reservation.StartAt
		if row.Type == NotificationEnd10 {
			expectedTarget, expectedExpiry = reservation.EndAt.Add(-10*time.Minute), reservation.EndAt
		}
		if row.ExpiresAt == nil || !row.ExpiresAt.Equal(expectedExpiry) || payload.TargetAt != expectedTarget.Unix() {
			return s.dao.FinishOutbox(ctx, row, dao.OutboxSuppressed, "reservation time changed", nil)
		}
		if !expectedExpiry.After(s.now()) {
			return s.dao.FinishOutbox(ctx, row, dao.OutboxSuppressed, "notification expired", nil)
		}
	}
	if row.Type == NotificationAway60 || row.Type == NotificationAway80 {
		// 旧 outbox 可能没有 expires_at，仍须依据载荷及预约快照拒绝过期提醒。
		if payload.ReservationID != row.ExternalReservationID || payload.EpisodeVersion <= 0 || payload.EndAt <= s.now().Unix() {
			return s.dao.FinishOutbox(ctx, row, dao.OutboxSuppressed, "away notification expired or invalid", nil)
		}
		reservation, err := s.dao.Reservation(ctx, row.StudentID, row.ExternalReservationID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return s.dao.FinishOutbox(ctx, row, dao.OutboxSuppressed, "reservation no longer active", nil)
		}
		if err != nil {
			return err
		}
		if terminalReservationStatus(reservation.Status) || !reservation.EndAt.After(s.now()) || reservation.EndAt.Unix() != payload.EndAt {
			return s.dao.FinishOutbox(ctx, row, dao.OutboxSuppressed, "reservation no longer active", nil)
		}
		if current, err := s.awayEpisodeCurrent(ctx, row.StudentID, row.ExternalReservationID, payload.EpisodeVersion); err != nil {
			return err
		} else if !current {
			return s.dao.FinishOutbox(ctx, row, dao.OutboxSuppressed, "away episode changed", nil)
		}
		token, err := s.libraryToken(ctx, row.StudentID)
		if err != nil {
			return err
		}
		current, err := s.crawler.GetCurrentReservation(ctx, token)
		if err != nil {
			return err
		}
		now := s.now()
		away := awayObservation{}
		state := dao.AwayStateEnded
		if current != nil && current.ID == row.ExternalReservationID && !terminalReservationStatus(current.Status) {
			_, end, err := current.Times()
			if err != nil {
				return err
			}
			if end.After(now) {
				away = currentAwayObservation(*current, now)
				state = dao.AwayStateReturned
			}
			if end.Unix() != payload.EndAt {
				return s.dao.FinishOutbox(ctx, row, dao.OutboxSuppressed, "reservation time changed", nil)
			}
		}
		if !away.isAway {
			ended, err := s.endAwayEpisode(ctx, row.StudentID, row.ExternalReservationID, row.PreferenceVersion, payload.EpisodeVersion, state)
			if err != nil {
				return err
			}
			if ended {
				return nil
			}
			return s.dao.FinishOutbox(ctx, row, dao.OutboxSuppressed, "away condition no longer holds", nil)
		}
		threshold := 60
		if row.Type == NotificationAway80 {
			threshold = 80
		}
		if away.elapsedKnown && away.elapsedMinutes < threshold {
			return s.dao.FinishOutbox(ctx, row, dao.OutboxSuppressed, "away condition no longer holds", nil)
		}
		// 上游复核期间可能已发生返回或偏好变更，投递前再次检查 claim、订阅及 episode。
		// 本地检查与远程 Publish 仍非原子操作，仅缩小并发状态变化的窗口。
		canSend, err := s.dao.CanSendOutbox(ctx, row)
		if err != nil {
			return err
		}
		if canSend {
			canSend, err = s.awayEpisodeCurrent(ctx, row.StudentID, row.ExternalReservationID, payload.EpisodeVersion)
			if err != nil {
				return err
			}
		}
		if !canSend {
			err := s.dao.FinishOutbox(ctx, row, dao.OutboxSuppressed, "subscription or away episode changed", nil)
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// 返回观察可能已抑制当前 sending，无需重复完成。
				return nil
			}
			return err
		}
		if payload.EndAt <= s.now().Unix() || (row.ExpiresAt != nil && !row.ExpiresAt.After(s.now())) {
			return s.dao.FinishOutbox(ctx, row, dao.OutboxSuppressed, "notification expired", nil)
		}
	}
	event := payloadFeedEvent(row.DedupeKey, payload)
	callCtx, cancel := s.remoteCallContext(ctx)
	result, publishErr := s.feed.Publish(callCtx, row.StudentID, event)
	cancel()
	if publishErr == nil {
		if result == PublishDuplicate && s.metrics != nil {
			s.metrics.NotificationDeduplicatedTotal.WithLabelValues(row.Type).Inc()
		}
		status := dao.OutboxSent
		if result == PublishSuppressed {
			status = dao.OutboxSuppressed
		}
		return s.dao.FinishOutbox(ctx, row, status, "", nil)
	}
	s.logger.WithContext(ctx).Warn("library reminder outbox publish failed",
		logger.Int64("outbox_id", row.ID),
		logger.String("notification_type", row.Type),
		logger.String("student_id", maskStudentID(row.StudentID)),
		logger.Int("attempt", row.Attempts),
		logger.Error(publishErr),
	)
	if row.Attempts >= s.config.RetryMaxAttempts {
		return s.dao.FinishOutbox(ctx, row, dao.OutboxFailed, "retry limit reached", nil)
	}
	next := s.now().Add(time.Duration(1<<min(row.Attempts, 8)) * time.Second)
	return s.dao.FinishOutbox(ctx, row, dao.OutboxFailed, "transient publish failure", &next)
}

// 单行处理失败时使用独立短上下文释放 sending 状态；即使释放失败，仍继续处理本批其余记录。
func (s *ReminderService) releaseOutboxAfterFailure(parent context.Context, row dao.NotificationOutbox) {
	lastError := "transient processing failure"
	var next *time.Time
	if row.Attempts < s.config.RetryMaxAttempts {
		retryAt := s.now().Add(time.Duration(1<<min(row.Attempts, 8)) * time.Second)
		next = &retryAt
	} else {
		lastError = "retry limit reached"
	}
	releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(parent), claimReleaseTimeout)
	defer cancel()
	if err := s.dao.FinishOutbox(releaseCtx, row, dao.OutboxFailed, lastError, next); err != nil {
		s.logger.WithContext(parent).Warn("library reminder outbox release failed",
			logger.Int64("outbox_id", row.ID),
			logger.Error(err),
		)
	}
}

func (s *ReminderService) releaseClaimedOutbox(parent context.Context, rows []dao.NotificationOutbox) {
	if len(rows) == 0 {
		return
	}
	releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(parent), claimReleaseTimeout)
	defer cancel()
	if err := s.dao.ReleaseClaimedOutbox(releaseCtx, rows); err != nil {
		s.logger.WithContext(parent).Warn("release claimed library reminder outbox failed", logger.Int("count", len(rows)), logger.Error(err))
	}
}

func (s *ReminderService) observeWorkMetrics(ctx context.Context) {
	if s.metrics == nil || s.dao == nil || ctx.Err() != nil {
		return
	}
	snapshot, err := s.dao.MetricsSnapshot(ctx, s.now(), s.config.RetryMaxAttempts)
	if err != nil {
		s.logger.WithContext(ctx).Warn("collect library reminder work metrics failed", logger.Error(err))
		return
	}
	s.metrics.NotificationJobs.Reset()
	for _, row := range snapshot.Jobs {
		s.metrics.NotificationJobs.WithLabelValues(row.Type, row.Status).Set(float64(row.Count))
	}
	s.metrics.Outbox.Reset()
	for _, row := range snapshot.Outbox {
		s.metrics.Outbox.WithLabelValues(row.Status).Set(float64(row.Count))
	}
	jobLag := 0.0
	if snapshot.OldestDueJobAt != nil {
		jobLag = max(0, s.now().Sub(*snapshot.OldestDueJobAt).Seconds())
	}
	s.metrics.NotificationJobLagSeconds.Set(jobLag)
	outboxAge := 0.0
	if snapshot.OldestOutboxAt != nil {
		outboxAge = max(0, s.now().Sub(*snapshot.OldestOutboxAt).Seconds())
	}
	s.metrics.OutboxOldestAgeSeconds.Set(outboxAge)
	s.metrics.ActiveReservations.Set(float64(snapshot.ActiveUsers))
}

func (s *ReminderService) scheduleJob(ctx context.Context, sub dao.LibraryReminderSubscription, notificationType, reservationID string, episode int, runAt, targetAt time.Time, expiresAt *time.Time, businessVersion int64) error {
	key := reminderJobKey(sub.StudentID, reservationID, notificationType, businessVersion)
	return s.dao.UpsertJob(ctx, &dao.NotificationJob{LogicalKey: key, StudentID: sub.StudentID, ExternalReservationID: reservationID, EpisodeVersion: episode, PreferenceVersion: sub.PreferenceVersion, Type: notificationType, TargetAt: targetAt, ExpiresAt: expiresAt, RunAt: runAt, Status: dao.JobPending, Version: 1})
}

func reminderJobKey(studentID, reservationID, notificationType string, businessVersion int64) string {
	return boundedDedupeKey(fmt.Sprintf("%s:%s:%s:%d", studentID, reservationID, notificationType, businessVersion))
}

func (s *ReminderService) notificationEnabled(notificationType string) bool {
	types := s.config.NotificationTypes
	if types == nil {
		return false
	}
	switch notificationType {
	case NotificationReservationDiscovered:
		return types.ReservationDiscovered
	case NotificationStart30:
		return types.Start30
	case NotificationEnd10:
		return types.End10
	case NotificationAway60:
		return types.Away60
	case NotificationAway80:
		return types.Away80
	case NotificationBreach:
		return types.Breach
	case NotificationBlacklisted:
		return types.Blacklisted
	default:
		return false
	}
}

func (s *ReminderService) enqueueFact(ctx context.Context, sub dao.LibraryReminderSubscription, notificationType, reservationID string, episode int, start, end, target time.Time) error {
	key := boundedDedupeKey(fmt.Sprintf("%s:%s:%s", sub.StudentID, reservationID, notificationType))
	reservation, _ := s.dao.Reservation(ctx, sub.StudentID, reservationID)
	payload := notificationPayload{NotificationType: notificationType, ReservationID: reservationID, StartAt: start.Unix(), EndAt: end.Unix(), TargetAt: target.Unix(), EpisodeVersion: episode}
	if reservation != nil {
		payload.SeatID, payload.SeatLabel, payload.Location = reservation.SeatID, reservation.SeatLabel, reservation.Location
	}
	return s.enqueuePayload(ctx, sub, key, notificationType, payload)
}

func (s *ReminderService) enqueuePayload(ctx context.Context, sub dao.LibraryReminderSubscription, key, notificationType string, payload notificationPayload) error {
	// 下游通知正文最多使用 8 KiB，入 outbox 前同步截断，避免 JSON 转义后超过 BLOB 上限。
	payload.Message = truncateUTF8(payload.Message, notificationMessageMaxBytes)
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return s.dao.EnqueueOutbox(ctx, &dao.NotificationOutbox{DedupeKey: key, StudentID: sub.StudentID, ExternalReservationID: payload.ReservationID, PreferenceVersion: sub.PreferenceVersion, Type: notificationType, Payload: raw, Status: dao.OutboxPending, NextAttemptAt: s.now()})
}

func (s *ReminderService) libraryToken(ctx context.Context, studentID string) (string, error) {
	callCtx, cancel := s.remoteCallContext(ctx)
	defer cancel()
	resp, err := s.user.GetLibrarySeatToken(callCtx, &userv1.GetLibraryTokenRequest{StudentId: studentID})
	if err != nil {
		return "", err
	}
	if resp == nil || resp.GetToken() == "" {
		return "", errors.New("user service returned an empty library token")
	}
	return resp.GetToken(), nil
}

func libraryTokenFailureStatus(err error) string {
	if userv1.IsIncorrectPasswordError(err) || userv1.IsUserNotFoundError(err) {
		return dao.SubscriptionAuthError
	}
	switch status.Code(err) {
	case codes.Unauthenticated, codes.PermissionDenied, codes.NotFound:
		return dao.SubscriptionAuthError
	default:
		return dao.SubscriptionAuthUpstreamUnknown
	}
}

func (s *ReminderService) remoteCallContext(parent context.Context) (context.Context, context.CancelFunc) {
	if s.config.RequestTimeout <= 0 {
		return context.WithCancel(parent)
	}
	return context.WithTimeout(parent, s.config.RequestTimeout)
}

func (s *ReminderService) subscriptionStillCurrent(ctx context.Context, expected dao.LibraryReminderSubscription) (bool, error) {
	current, err := s.dao.Subscription(ctx, expected.StudentID)
	if err != nil {
		return false, err
	}
	return current.Enabled && current.PreferenceVersion == expected.PreferenceVersion, nil
}

// refreshFailure 为单用户刷新失败附加低基数阶段（stage）与错误分类（kind）信息。
type refreshFailure struct {
	stage string
	kind  string
	err   error
}

func (e *refreshFailure) Error() string {
	return strings.ReplaceAll(e.stage, "_", " ") + ": " + e.err.Error()
}

func (e *refreshFailure) Unwrap() error { return e.err }

// stageFailure 包装刷新链路错误，仅当父操作本身已取消或过期时才覆盖分类。
// 上游请求自身的超时（remoteCallContext 的 8 秒上限）错误链同样匹配
// context.DeadlineExceeded/Canceled，但父 ctx 仍存活，必须保留 upstreamFailureKind
// 或阶段固有的分类，否则 upstream_timeout 不可达，无法与调度任务的 25 分钟
// 截止或关闭取消区分。
func stageFailure(ctx context.Context, err error, stage, kind string) error {
	switch ctx.Err() {
	case context.DeadlineExceeded:
		kind = "context_deadline"
	case context.Canceled:
		kind = "context_canceled"
	}
	return &refreshFailure{stage: stage, kind: kind, err: err}
}

// upstreamFailureKind 将学校上游错误映射为批量错误的低基数分类，
// 复用 crawler 导出的上游错误分类结果。
func upstreamFailureKind(err error) string {
	switch crawler.ClassifyUpstreamError(err) {
	case "timeout":
		return "upstream_timeout"
	case "auth_error":
		return "upstream_auth"
	case "http_error":
		return "upstream_http"
	case "business_error":
		return "upstream_business"
	case "network_or_decode_error":
		return "upstream_http"
	default:
		return "invalid_response"
	}
}

// gormDatabaseErrors 为可归为 database 分类的 gorm 常规错误，
// 供 fallback 分类未附着 refreshFailure 信息的普通数据库失败。
var gormDatabaseErrors = []error{
	gorm.ErrRecordNotFound,
	gorm.ErrInvalidTransaction,
	gorm.ErrNotImplemented,
	gorm.ErrMissingWhereClause,
	gorm.ErrUnsupportedRelation,
	gorm.ErrPrimaryKeyRequired,
	gorm.ErrModelValueRequired,
	gorm.ErrModelAccessibleFieldsRequired,
	gorm.ErrSubQueryRequired,
	gorm.ErrInvalidData,
	gorm.ErrUnsupportedDriver,
	gorm.ErrRegistered,
	gorm.ErrInvalidField,
	gorm.ErrEmptySlice,
	gorm.ErrDryRunModeUnsupported,
	gorm.ErrInvalidDB,
	gorm.ErrInvalidValue,
	gorm.ErrInvalidValueOfLength,
	gorm.ErrPreloadNotAllowed,
	gorm.ErrDuplicatedKey,
	gorm.ErrForeignKeyViolated,
	gorm.ErrCheckConstraintViolated,
}

// classifyFallbackFailure 为未附着阶段信息的错误给出低基数分类。
func classifyFallbackFailure(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "context_deadline"
	case errors.Is(err, context.Canceled):
		return "context_canceled"
	}
	for _, dbErr := range gormDatabaseErrors {
		if errors.Is(err, dbErr) {
			return "database"
		}
	}
	return "unknown"
}

// refreshFailureInfo 从错误中提取阶段与分类；未附着 refreshFailure 时给出回退分类。
func refreshFailureInfo(err error) (stage, kind string) {
	var failure *refreshFailure
	if errors.As(err, &failure) {
		return failure.stage, failure.kind
	}
	return "", classifyFallbackFailure(err)
}

const batchErrorSampleLimit = 20

// subscriptionFailure 保留批量刷新中的单条失败样例，StudentID 必须已脱敏。
// ErrText 只保存固定上限的截断文本；原始错误仅保留一份在 batch.Cause。
type subscriptionFailure struct {
	StudentID string
	Stage     string
	ErrText   string
}

// subscriptionBatchError 汇总批量订阅刷新的结果与分类计数，避免用 errors.Join 拼接
// 全部用户错误导致错误文本与内存无限增长；通过 Unwrap 暴露代表性 Cause，
// 使 errors.Is(err, context.DeadlineExceeded) 仍然可用。
type subscriptionBatchError struct {
	Total     int
	Launched  int
	Succeeded int
	Failed    int
	Canceled  int
	Groups    map[string]int
	Samples   []subscriptionFailure
	// Cause 保留首个失败的原始错误（不受样例截断影响），供 Unwrap 保持 errors.Is 语义。
	Cause error
}

func (e *subscriptionBatchError) Error() string {
	return fmt.Sprintf("subscription batch failed: total=%d launched=%d succeeded=%d failed=%d canceled=%d", e.Total, e.Launched, e.Succeeded, e.Failed, e.Canceled)
}

func (e *subscriptionBatchError) Unwrap() error { return e.Cause }

// SampleLogs 输出有限的脱敏错误样例，便于结构化日志。
func (e *subscriptionBatchError) SampleLogs() []map[string]string {
	samples := make([]map[string]string, 0, len(e.Samples))
	for _, sample := range e.Samples {
		samples = append(samples, map[string]string{
			"student_id": sample.StudentID,
			"stage":      sample.Stage,
			"error":      sample.ErrText,
		})
	}
	return samples
}

// LogFields 将批量结果转换为 scheduler 最终任务失败日志的结构化字段。
func (e *subscriptionBatchError) LogFields() []logger.Field {
	return []logger.Field{
		logger.Int("total", e.Total),
		logger.Int("launched", e.Launched),
		logger.Int("succeeded", e.Succeeded),
		logger.Int("failed", e.Failed),
		logger.Int("canceled", e.Canceled),
		logger.Any("error_groups", e.Groups),
		logger.Any("error_samples", e.SampleLogs()),
	}
}

// forEachSubscription 并发执行用户任务并汇总批量结果。任何用户失败都会保留
// 分类计数与有限数量的脱敏样例，并通过 subscriptionBatchError 向上返回，
// 不再丢弃其他用户的失败原因。
func (s *ReminderService) forEachSubscription(ctx context.Context, rows []dao.LibraryReminderSubscription, fn func(context.Context, dao.LibraryReminderSubscription) error) error {
	limit := s.config.UserConcurrency
	if limit <= 0 {
		limit = 1
	}
	batch := &subscriptionBatchError{Total: len(rows), Groups: map[string]int{}}
	sem := make(chan struct{}, limit)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var launchErr error

launch:
	for i := range rows {
		if err := ctx.Err(); err != nil {
			launchErr = err
			break
		}
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			// 必须退出外层循环：仅跳出 select 会继续启动未取得 token 的
			// goroutine，其 defer 的 <-sem 将永久阻塞，wg.Wait() 无法返回。
			launchErr = ctx.Err()
			break launch
		}
		row := rows[i]
		batch.Launched++
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			if err := fn(ctx, row); err != nil {
				stage, kind := refreshFailureInfo(err)
				mu.Lock()
				batch.Failed++
				batch.Groups[kind]++
				if batch.Cause == nil {
					// 代表性失败：仅用于 errors.Is 判断，不展开全部用户错误。
					batch.Cause = err
				}
				if len(batch.Samples) < batchErrorSampleLimit {
					batch.Samples = append(batch.Samples, subscriptionFailure{StudentID: maskStudentID(row.StudentID), Stage: stage, ErrText: truncateUTF8(err.Error(), subscriptionErrorSampleMaxBytes)})
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	batch.Canceled = batch.Total - batch.Launched
	batch.Succeeded = batch.Launched - batch.Failed
	if launchErr == nil && batch.Failed == 0 {
		return nil
	}
	if launchErr != nil {
		// 取消/超时优先：覆盖首个失败的 Cause，避免 errors.Is 误判为具体用户错误。
		batch.Cause = launchErr
	}
	return batch
}

// runUserOperation 先获取一次用户任务许可，再在许可范围内完成全部重试，
// 避免首次失败设置的最小间隔将同一次操作的后续重试短路。
func (s *ReminderService) runUserOperation(ctx context.Context, row dao.LibraryReminderSubscription, taskType string, minInterval time.Duration, fn func(context.Context, dao.LibraryReminderSubscription) error) (err error) {
	finish, started := s.userTaskGate.start(row.StudentID, taskType, row.PreferenceVersion, s.now(), minInterval)
	if !started {
		return nil
	}
	defer func() { finish(err == nil, s.now()) }()

	attempts := s.config.UpstreamRetryAttempts
	if attempts <= 0 {
		attempts = 1
	}
	for attempt := 0; attempt < attempts; attempt++ {
		delay := time.Duration(0)
		if attempt > 0 {
			delay = time.Duration(1<<min(attempt-1, 5)) * 250 * time.Millisecond
		}
		if s.config.UserJitter > 0 {
			delay += time.Duration(rand.Int63n(int64(s.config.UserJitter) + 1))
		}
		if delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
		err = fn(ctx, row)
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	return err
}

func terminalReservationStatus(status string) bool {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "CANCEL", "STOP", "FINISH", "LEAVE_EARLY", "MISS":
		return true
	default:
		return false
	}
}

func payloadFeedEvent(dedupeKey string, payload notificationPayload) *feedv1.FeedEvent {
	title, content := "图书馆提醒", "你的图书馆预约状态有更新。"
	switch payload.NotificationType {
	case NotificationReservationDiscovered:
		content = fmt.Sprintf("检测到你预约了%s %s。", payload.Location, payload.SeatLabel)
	case NotificationStart30:
		content = fmt.Sprintf("你预约的座位 %s 即将开始。", payload.SeatLabel)
	case NotificationEnd10:
		content = fmt.Sprintf("你预约的座位 %s 即将结束。", payload.SeatLabel)
	case NotificationAway60:
		content = "你已暂离约 60 分钟，请及时返回。"
	case NotificationAway80:
		content = "你已暂离约 80 分钟，请尽快返回。"
	case NotificationBreach:
		content = "检测到新的图书馆违约记录，请查看图书馆规则。"
	case NotificationBlacklisted:
		content = truncateUTF8(payload.Message, notificationMessageMaxBytes)
		if content == "" {
			content = "检测到图书馆预约权限受限，请查看图书馆提示。"
		}
	}
	extend := map[string]string{"notification_type": payload.NotificationType}
	set := func(key, value string) {
		if value != "" {
			extend[key] = value
		}
	}
	set("reservation_id", payload.ReservationID)
	set("seat_id", payload.SeatID)
	set("seat_label", payload.SeatLabel)
	set("location", payload.Location)
	if payload.StartAt != 0 {
		extend["start_at"] = strconvFormat(payload.StartAt)
	}
	if payload.EndAt != 0 {
		extend["end_at"] = strconvFormat(payload.EndAt)
	}
	if payload.TargetAt != 0 {
		extend["target_at"] = strconvFormat(payload.TargetAt)
	}
	if payload.EpisodeVersion != 0 {
		extend["episode_version"] = fmt.Sprint(payload.EpisodeVersion)
	}
	return &feedv1.FeedEvent{Type: feedv1.FeedEventType_LIBRARY, Title: title, Content: content, ExtendFields: extend, DedupeKey: dedupeKey, Source: "library", OccurredAt: payload.TargetAt}
}

func strconvFormat(value int64) string { return fmt.Sprintf("%d", value) }

func boundedDedupeKey(value string) string {
	if len(value) <= 255 {
		return value
	}
	sum := sha256.Sum256([]byte(value))
	return "library:sha256:" + hex.EncodeToString(sum[:])
}

func truncateUTF8(value string, maxBytes int) string {
	if len(value) <= maxBytes {
		return value
	}
	value = value[:maxBytes]
	for len(value) > 0 && !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return value
}

func maskStudentID(value string) string {
	if len(value) <= 4 {
		return "****"
	}
	return value[:2] + "****" + value[len(value)-2:]
}
