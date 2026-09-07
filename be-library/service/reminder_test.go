package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/asynccnu/ccnubox-be/be-library/conf"
	"github.com/asynccnu/ccnubox-be/be-library/crawler"
	"github.com/asynccnu/ccnubox-be/be-library/repository/dao"
	feedv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feed/v1"
	userv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/user/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger/zapx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type reminderTestFeed struct {
	FeedGateway
	changes   func(context.Context, int64, int32) ([]LibraryPreferenceChange, int64, error)
	published int
}

func (f *reminderTestFeed) PreferenceChanges(ctx context.Context, after int64, limit int32) ([]LibraryPreferenceChange, int64, error) {
	return f.changes(ctx, after, limit)
}

func (f *reminderTestFeed) Publish(context.Context, string, *feedv1.FeedEvent) (PublishResult, error) {
	f.published++
	return PublishAccepted, nil
}

type reminderTestUser struct{ userv1.UserServiceClient }

func (reminderTestUser) GetLibrarySeatToken(context.Context, *userv1.GetLibraryTokenRequest, ...grpc.CallOption) (*userv1.GetLibraryTokenResponse, error) {
	return &userv1.GetLibraryTokenResponse{Token: "test-token"}, nil
}

type reminderTestCrawler struct {
	crawler.ReminderCrawler
	current *crawler.ReminderReservation
}

func (c reminderTestCrawler) GetCurrentReservation(context.Context, string) (*crawler.ReminderReservation, error) {
	return c.current, nil
}

func newReminderTestService(t *testing.T) (*ReminderService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "reminder.db")), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&dao.LibraryReminderSubscription{}, &dao.LibraryPreferenceSyncCursor{}, &dao.ReservationSnapshot{}, &dao.AwayEpisode{}, &dao.NotificationJob{}, &dao.NotificationOutbox{}); err != nil {
		t.Fatal(err)
	}
	config := (*conf.ServerConf)(nil).Reminder()
	config.Enabled = true
	no := false
	config.DryRun, config.BaselineOnEnable = &no, &no
	s := &ReminderService{dao: dao.NewReminderDAO(db), config: config, user: reminderTestUser{}, feed: &reminderTestFeed{}, logger: zapx.NewZapLogger(zap.NewNop())}
	now := time.Date(2026, 6, 1, 4, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	return s, db
}

func seedReminderReservation(t *testing.T, s *ReminderService, db *gorm.DB) (dao.LibraryReminderSubscription, crawler.ReminderReservation, dao.AwayEpisode) {
	t.Helper()
	sub := dao.LibraryReminderSubscription{StudentID: "20260001", Enabled: true, PreferenceVersion: 2, BaselineCompleted: true}
	if err := db.Create(&sub).Error; err != nil {
		t.Fatal(err)
	}
	current := crawler.ReminderReservation{ID: "reservation-1", Status: "USING", MakeDateStr: "2026-06-01", MakeBegin: 10 * 60, MakeEnd: 14 * 60, AwayTimeM: 65}
	start, end, err := current.Times()
	if err != nil {
		t.Fatal(err)
	}
	snapshot := dao.ReservationSnapshot{StudentID: sub.StudentID, ExternalReservationID: current.ID, StartAt: start, EndAt: end, Status: current.Status, FirstSeenAt: s.now().Add(-time.Hour), LastSeenAt: s.now().Add(-time.Hour)}
	if _, err := s.dao.SaveReservation(context.Background(), &snapshot); err != nil {
		t.Fatal(err)
	}
	episode := dao.AwayEpisode{StudentID: sub.StudentID, ExternalReservationID: current.ID, EpisodeVersion: 2, State: dao.AwayStateAway, AwayStartedAt: s.now().Add(-65 * time.Minute)}
	if err := s.dao.SaveAwayEpisode(context.Background(), &episode); err != nil {
		t.Fatal(err)
	}
	s.crawler = reminderTestCrawler{current: &current}
	return sub, current, episode
}

func seedReminderWork(t *testing.T, s *ReminderService, db *gorm.DB, sub dao.LibraryReminderSubscription, current crawler.ReminderReservation, notificationType string) (dao.NotificationJob, dao.NotificationOutbox) {
	t.Helper()
	start, end, err := current.Times()
	if err != nil {
		t.Fatal(err)
	}
	payload := notificationPayload{NotificationType: notificationType, ReservationID: current.ID, StartAt: start.Unix(), EndAt: end.Unix(), TargetAt: s.now().Unix(), EpisodeVersion: 2}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	job := dao.NotificationJob{LogicalKey: notificationType, StudentID: sub.StudentID, ExternalReservationID: current.ID, EpisodeVersion: 2, PreferenceVersion: sub.PreferenceVersion, Type: notificationType, TargetAt: s.now(), RunAt: s.now(), ExpiresAt: &end, Status: dao.JobRunning, Version: 3, Attempts: 1}
	row := dao.NotificationOutbox{DedupeKey: notificationType, StudentID: sub.StudentID, ExternalReservationID: current.ID, PreferenceVersion: sub.PreferenceVersion, Type: notificationType, Payload: raw, Status: dao.OutboxSending, Attempts: 1, NextAttemptAt: s.now(), ExpiresAt: &end}
	if err := db.Create(&job).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	return job, row
}

func TestEndAwayEpisodeChecksVersions(t *testing.T) {
	for _, tc := range []struct {
		name              string
		preferenceVersion int64
		episodeVersion    int
		disabled          bool
		wantEnded         bool
	}{
		{name: "current", preferenceVersion: 2, episodeVersion: 2, wantEnded: true},
		{name: "old_episode", preferenceVersion: 2, episodeVersion: 1},
		{name: "old_preference", preferenceVersion: 1, episodeVersion: 2},
		{name: "disabled", preferenceVersion: 2, episodeVersion: 2, disabled: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, db := newReminderTestService(t)
			sub, current, episode := seedReminderReservation(t, s, db)
			seedReminderWork(t, s, db, sub, current, NotificationAway60)
			seedReminderWork(t, s, db, sub, current, NotificationAway80)
			if tc.disabled {
				if err := db.Model(&sub).Update("enabled", false).Error; err != nil {
					t.Fatal(err)
				}
			}
			ended, err := s.endAwayEpisode(context.Background(), sub.StudentID, current.ID, tc.preferenceVersion, tc.episodeVersion, dao.AwayStateReturned)
			if err != nil || ended != tc.wantEnded {
				t.Fatalf("ended=%v err=%v, want=%v", ended, err, tc.wantEnded)
			}
			if err := db.First(&episode, episode.ID).Error; err != nil {
				t.Fatal(err)
			}
			wantState, wantJob, wantOutbox := dao.AwayStateAway, dao.JobRunning, dao.OutboxSending
			if tc.wantEnded {
				wantState, wantJob, wantOutbox = dao.AwayStateReturned, dao.JobCancelled, dao.OutboxSuppressed
			}
			if episode.State != wantState || episode.EpisodeVersion != 2 {
				t.Fatalf("episode=%+v, want state=%s version=2", episode, wantState)
			}
			assertReminderWorkStatus(t, db, wantJob, wantOutbox, 2)
		})
	}
}

func assertReminderWorkStatus(t *testing.T, db *gorm.DB, jobStatus, outboxStatus string, wantCount int64) {
	t.Helper()
	var jobs, outbox int64
	if err := db.Model(&dao.NotificationJob{}).Where("status = ?", jobStatus).Count(&jobs).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&dao.NotificationOutbox{}).Where("status = ?", outboxStatus).Count(&outbox).Error; err != nil {
		t.Fatal(err)
	}
	if jobs != wantCount || outbox != wantCount {
		t.Fatalf("jobs(%s)=%d outbox(%s)=%d, want=%d", jobStatus, jobs, outboxStatus, outbox, wantCount)
	}
}

func TestApplyActiveObservationPersistsInactiveReservation(t *testing.T) {
	for _, tc := range []struct {
		status string
		end    crawler.Minute
	}{
		{status: "CANCEL", end: 14 * 60},
		{status: "STOP", end: 14 * 60},
		{status: "FINISH", end: 14 * 60},
		{status: " leave_early ", end: 14 * 60},
		{status: "MISS", end: 14 * 60},
		{status: "USING", end: 12 * 60},
		{status: "USING", end: 12*60 - 1},
	} {
		t.Run(fmt.Sprintf("%s_%d", tc.status, tc.end), func(t *testing.T) {
			s, db := newReminderTestService(t)
			sub, current, episode := seedReminderReservation(t, s, db)
			for _, typ := range []string{NotificationStart30, NotificationEnd10, NotificationAway60, NotificationAway80} {
				seedReminderWork(t, s, db, sub, current, typ)
			}
			current.Status, current.MakeEnd = tc.status, tc.end
			err := s.dao.Transaction(context.Background(), func(txDAO *dao.ReminderDAO) error {
				locked, err := txDAO.SubscriptionForUpdate(context.Background(), sub.StudentID)
				if err != nil {
					return err
				}
				txService := *s
				txService.dao = txDAO
				return txService.applyActiveObservation(context.Background(), *locked, &current, s.now())
			})
			if err != nil {
				t.Fatal(err)
			}
			snapshot, err := s.dao.Reservation(context.Background(), sub.StudentID, current.ID)
			_, end, _ := current.Times()
			if err != nil || snapshot.Status != strings.ToUpper(strings.TrimSpace(tc.status)) || !snapshot.EndAt.Equal(end) || !snapshot.LastSeenAt.Equal(s.now()) {
				t.Fatalf("snapshot=%+v err=%v", snapshot, err)
			}
			if err := db.First(&episode, episode.ID).Error; err != nil || episode.State != dao.AwayStateEnded {
				t.Fatalf("episode=%+v err=%v", episode, err)
			}
			assertReminderWorkStatus(t, db, dao.JobCancelled, dao.OutboxSuppressed, 4)
			active, err := s.dao.ActiveSubscriptions(context.Background(), s.now(), 10)
			if err != nil || len(active) != 0 {
				t.Fatalf("active=%+v err=%v", active, err)
			}
			storedSub, err := s.dao.Subscription(context.Background(), sub.StudentID)
			if err != nil || storedSub.LastActiveScanAt == nil || !storedSub.LastActiveScanAt.Equal(s.now()) {
				t.Fatalf("subscription=%+v err=%v", storedSub, err)
			}
		})
	}
}

func TestSendAwayOutboxRechecksEpisodeBeforePublish(t *testing.T) {
	for _, change := range []string{"none", "returned", "new_episode", "query_error"} {
		t.Run(change, func(t *testing.T) {
			s, db := newReminderTestService(t)
			sub, current, episode := seedReminderReservation(t, s, db)
			_, row := seedReminderWork(t, s, db, sub, current, NotificationAway60)
			checks := 0
			queryErr := errors.New("episode query failed")
			// 在最终 claim 查询已读出可发送后注入扫描提交，固定复现复核与发送的交错。
			if err := db.Callback().Query().After("gorm:query").Register("test:change_episode", func(tx *gorm.DB) {
				if tx.Statement.Table != "notification_outbox" {
					return
				}
				if _, ok := tx.Statement.Dest.(*int64); !ok {
					return
				}
				checks++
				if checks != 2 {
					return
				}
				switch change {
				case "returned":
					ended, err := s.endAwayEpisode(context.Background(), sub.StudentID, current.ID, sub.PreferenceVersion, episode.EpisodeVersion, dao.AwayStateReturned)
					if err != nil || !ended {
						t.Fatalf("end episode: ended=%v err=%v", ended, err)
					}
				case "new_episode":
					episode.ID = 0
					episode.EpisodeVersion++
					if err := s.dao.SaveAwayEpisode(context.Background(), &episode); err != nil {
						t.Fatal(err)
					}
				}
			}); err != nil {
				t.Fatal(err)
			}
			if err := db.Callback().Query().Before("gorm:query").Register("test:episode_error", func(tx *gorm.DB) {
				if change == "query_error" && checks == 2 && tx.Statement.Table == "away_episodes" {
					tx.AddError(queryErr)
				}
			}); err != nil {
				t.Fatal(err)
			}
			err := s.sendOutboxRow(context.Background(), row)
			if change == "query_error" {
				if !errors.Is(err, queryErr) {
					t.Fatalf("err=%v, want=%v", err, queryErr)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			wantPublished, wantStatus := 0, dao.OutboxSuppressed
			if change == "none" {
				wantPublished, wantStatus = 1, dao.OutboxSent
			} else if change == "query_error" {
				wantStatus = dao.OutboxSending
			}
			if checks != 2 || s.feed.(*reminderTestFeed).published != wantPublished {
				t.Fatalf("claim checks=%d published=%d, want checks=2 published=%d", checks, s.feed.(*reminderTestFeed).published, wantPublished)
			}
			if err := db.First(&row, row.ID).Error; err != nil || row.Status != wantStatus {
				t.Fatalf("outbox=%+v err=%v, want status=%s", row, err, wantStatus)
			}
		})
	}
}

func TestSendReservationOutboxRejectsTerminalSnapshot(t *testing.T) {
	for _, typ := range []string{NotificationStart30, NotificationEnd10} {
		t.Run(typ, func(t *testing.T) {
			s, db := newReminderTestService(t)
			sub, current, _ := seedReminderReservation(t, s, db)
			start, end, _ := current.Times()
			now := start.Add(-20 * time.Minute)
			s.now = func() time.Time { return now }
			_, row := seedReminderWork(t, s, db, sub, current, typ)
			target, expiry := start.Add(-30*time.Minute), start
			if typ == NotificationEnd10 {
				target, expiry = end.Add(-10*time.Minute), end
			}
			payload := notificationPayload{NotificationType: typ, ReservationID: current.ID, StartAt: start.Unix(), EndAt: end.Unix(), TargetAt: target.Unix()}
			var err error
			row.Payload, err = json.Marshal(payload)
			if err != nil {
				t.Fatal(err)
			}
			row.ExpiresAt = &expiry
			if err := db.Save(&row).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Model(&dao.ReservationSnapshot{}).Where("student_id = ? AND external_reservation_id = ?", sub.StudentID, current.ID).Update("status", "CANCEL").Error; err != nil {
				t.Fatal(err)
			}
			if err := s.sendOutboxRow(context.Background(), row); err != nil {
				t.Fatal(err)
			}
			if s.feed.(*reminderTestFeed).published != 0 {
				t.Fatal("published a cancelled reservation reminder")
			}
			if err := db.First(&row, row.ID).Error; err != nil || row.Status != dao.OutboxSuppressed || row.LastError != "reservation no longer active" {
				t.Fatalf("outbox=%+v err=%v", row, err)
			}
		})
	}
}

func TestDispatchAwayJobPersistsRetryExpiry(t *testing.T) {
	for _, expiry := range []string{"missing", "too_late", "earlier"} {
		t.Run(expiry, func(t *testing.T) {
			s, db := newReminderTestService(t)
			sub, current, _ := seedReminderReservation(t, s, db)
			current.AwayTimeM = 50
			s.crawler = reminderTestCrawler{current: &current}
			job, row := seedReminderWork(t, s, db, sub, current, NotificationAway60)
			if err := db.Delete(&row).Error; err != nil {
				t.Fatal(err)
			}
			_, end, _ := current.Times()
			wantExpiry := end
			switch expiry {
			case "missing":
				job.ExpiresAt = nil
			case "too_late":
				later := end.Add(time.Hour)
				job.ExpiresAt = &later
			case "earlier":
				wantExpiry = end.Add(-time.Hour)
				job.ExpiresAt = &wantExpiry
			}
			if err := db.Model(&job).Update("expires_at", job.ExpiresAt).Error; err != nil {
				t.Fatal(err)
			}
			if err := s.dispatchJob(context.Background(), job); err != nil {
				t.Fatal(err)
			}
			stored := dao.NotificationJob{}
			if err := db.First(&stored, job.ID).Error; err != nil {
				t.Fatal(err)
			}
			if stored.Status != dao.JobPending || !stored.RunAt.Equal(s.now().Add(10*time.Minute)) || stored.ExpiresAt == nil || !stored.ExpiresAt.Equal(wantExpiry) {
				t.Fatalf("job=%+v, want pending at +10m expiry=%v", stored, wantExpiry)
			}
			// 已重排的 claim 不能再次完成并覆盖已持久化的有效期。
			later := end.Add(2 * time.Hour)
			job.ExpiresAt = &later
			next := s.now().Add(time.Minute)
			if err := s.dao.FinishJob(context.Background(), job, dao.JobPending, "", &next); !errors.Is(err, gorm.ErrRecordNotFound) {
				t.Fatalf("stale claim err=%v", err)
			}
			if err := db.First(&stored, job.ID).Error; err != nil || !stored.ExpiresAt.Equal(wantExpiry) {
				t.Fatalf("stale claim changed expiry: job=%+v err=%v", stored, err)
			}
		})
	}
}

func TestSyncPreferencesSkipsInvalidStudentIDs(t *testing.T) {
	s, db := newReminderTestService(t)
	ctx := context.Background()
	if _, err := s.dao.ApplyPreferenceChanges(ctx, nil, 0); err != nil {
		t.Fatal(err)
	}
	pages := [][]LibraryPreferenceChange{
		{{Revision: 1, StudentID: "", Enabled: true}, {Revision: 2, StudentID: "20260001", Enabled: true}, {Revision: 3, StudentID: " 20260002", Enabled: true}},
		{{Revision: 4, StudentID: strings.Repeat("x", 65), Enabled: true}, {Revision: 5, StudentID: string([]byte{0xff}), Enabled: true}},
		{{Revision: 6, StudentID: "20260003", Enabled: true}},
	}
	wantCursors := []int64{0, 3, 5, 6}
	calls := 0
	s.feed = &reminderTestFeed{changes: func(_ context.Context, after int64, _ int32) ([]LibraryPreferenceChange, int64, error) {
		if calls >= len(wantCursors) || after != wantCursors[calls] {
			t.Fatalf("call=%d cursor=%d, want=%v", calls, after, wantCursors)
		}
		calls++
		if calls > len(pages) {
			return nil, after, nil
		}
		page := pages[calls-1]
		return page, page[len(page)-1].Revision, nil
	}}
	if err := s.syncPreferences(ctx); err != nil {
		t.Fatal(err)
	}
	cursor, err := s.dao.Cursor(ctx)
	if err != nil || cursor != 6 || calls != 4 {
		t.Fatalf("cursor=%d calls=%d err=%v", cursor, calls, err)
	}
	var subs []dao.LibraryReminderSubscription
	if err := db.Order("student_id").Find(&subs).Error; err != nil {
		t.Fatal(err)
	}
	if len(subs) != 2 || subs[0].StudentID != "20260001" || subs[0].FeedRevision != 2 || subs[1].StudentID != "20260003" || subs[1].FeedRevision != 6 || !subs[0].Enabled || !subs[1].Enabled {
		t.Fatalf("subscriptions=%+v", subs)
	}
}

func TestAwayEpisodeCurrentRejectsLegacyWork(t *testing.T) {
	for _, tc := range []struct {
		name    string
		version int
		missing bool
	}{
		{name: "zero_version", version: 0},
		{name: "negative_version", version: -1},
		{name: "missing_episode", version: 2, missing: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, db := newReminderTestService(t)
			sub, current, episode := seedReminderReservation(t, s, db)
			if tc.missing {
				if err := db.Delete(&episode).Error; err != nil {
					t.Fatal(err)
				}
			} else if err := db.Model(&episode).Update("episode_version", tc.version).Error; err != nil {
				t.Fatal(err)
			}
			ok, err := s.awayEpisodeCurrent(context.Background(), sub.StudentID, current.ID, tc.version)
			if err != nil || ok {
				t.Fatalf("current=%v err=%v", ok, err)
			}
		})
	}
}
