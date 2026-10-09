package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type reminderTestFeed struct {
	FeedGateway
	changes   func(context.Context, int64, int32) ([]LibraryPreferenceChange, int64, error)
	published int
	users     func(context.Context, int64, int64, int32) ([]LibraryReminderUser, int64, int64, error)
}

func (f *reminderTestFeed) PreferenceChanges(ctx context.Context, after int64, limit int32) ([]LibraryPreferenceChange, int64, error) {
	return f.changes(ctx, after, limit)
}

func (f *reminderTestFeed) Publish(context.Context, string, *feedv1.FeedEvent) (PublishResult, error) {
	f.published++
	return PublishAccepted, nil
}

type reminderTestUser struct{ userv1.UserServiceClient }

func (reminderTestUser) GetLibraryDiscussionToken(context.Context, *userv1.GetLibraryTokenRequest, ...grpc.CallOption) (*userv1.GetLibraryTokenResponse, error) {
	return &userv1.GetLibraryTokenResponse{Token: "discussion-token"}, nil
}

func (reminderTestUser) GetLibrarySeatToken(context.Context, *userv1.GetLibraryTokenRequest, ...grpc.CallOption) (*userv1.GetLibraryTokenResponse, error) {
	return &userv1.GetLibraryTokenResponse{Token: "test-token"}, nil
}

type reminderTestCrawler struct {
	crawler.ReminderCrawler
	current *crawler.ReminderReservation
	team    *crawler.ReminderTeam
	teamErr error
}

func (c reminderTestCrawler) GetCurrentTeam(context.Context, string) (*crawler.ReminderTeam, error) {
	return c.team, c.teamErr
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
	if err := db.AutoMigrate(&dao.LibraryReminderSubscription{}, &dao.LibraryPreferenceSyncCursor{}, &dao.ReservationSnapshot{}, &dao.LibraryTeamSnapshot{}, &dao.LibraryTeamInvitation{}, &dao.AwayEpisode{}, &dao.NotificationJob{}, &dao.NotificationOutbox{}); err != nil {
		t.Fatal(err)
	}
	config := (*conf.ServerConf)(nil).Reminder()
	config.Enabled = true
	no := false
	config.DryRun, config.BaselineOnEnable = &no, &no
	s := &ReminderService{invitation: newInvitationState(config.TeamInvitation.MaxConcurrentRequests), dao: dao.NewReminderDAO(db), config: config, user: reminderTestUser{}, feed: &reminderTestFeed{}, logger: zapx.NewZapLogger(zap.NewNop())}
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

func TestTeamSuccessBaselineAndDurableDedupe(t *testing.T) {
	ctx := context.Background()
	s, db := newReminderTestService(t)
	yes := true
	s.config.BaselineOnEnable = &yes
	s.config.NotificationTypes.TeamSuccess = true
	s.userTaskGate = newUserTaskGate()
	sub := dao.LibraryReminderSubscription{StudentID: "20260001", Enabled: true, PreferenceVersion: 1}
	if err := db.Create(&sub).Error; err != nil {
		t.Fatal(err)
	}
	observe := func(team *crawler.ReminderTeam) {
		t.Helper()
		s.crawler = reminderTestCrawler{team: team}
		if err := s.scanTeamUser(ctx, sub); err != nil {
			t.Fatal(err)
		}
		next := s.now().Add(5 * time.Minute)
		s.now = func() time.Time { return next }
	}
	observe(&crawler.ReminderTeam{ID: "team-1", Status: 1})
	var row dao.LibraryTeamSnapshot
	if err := db.First(&row).Error; err != nil || row.SuccessDisposition != "baseline" {
		t.Fatalf("baseline row=%+v err=%v", row, err)
	}
	observe(&crawler.ReminderTeam{ID: "team-2", Status: 0})
	observe(&crawler.ReminderTeam{ID: "team-2", Status: 1, OnDate: "2026-06-02"})
	observe(nil)
	observe(&crawler.ReminderTeam{ID: "team-2", Status: 1})
	var outbox []dao.NotificationOutbox
	if err := db.Find(&outbox).Error; err != nil || len(outbox) != 1 {
		t.Fatalf("outbox=%+v err=%v", outbox, err)
	}
	if outbox[0].DedupeKey != "library:team_success:20260001:team-2" || outbox[0].ExternalReservationID != "" {
		t.Fatalf("invalid outbox: %+v", outbox[0])
	}
	var payload notificationPayload
	if err := json.Unmarshal(outbox[0].Payload, &payload); err != nil || payload.TeamID != "team-2" || payload.OnDate != "2026-06-02" {
		t.Fatalf("payload=%+v err=%v", payload, err)
	}
	// Outbox 清理后，同队再次成功也不能重建消息。
	if err := db.Delete(&dao.NotificationOutbox{}, outbox[0].ID).Error; err != nil {
		t.Fatal(err)
	}
	observe(&crawler.ReminderTeam{ID: "team-2", Status: 1})
	var count int64
	db.Model(&dao.NotificationOutbox{}).Count(&count)
	if count != 0 {
		t.Fatalf("outbox recreated: %d", count)
	}
}

func TestTeamScanFailureDoesNotCompleteBaseline(t *testing.T) {
	ctx := context.Background()
	s, db := newReminderTestService(t)
	yes := true
	s.config.BaselineOnEnable = &yes
	s.config.NotificationTypes.TeamSuccess = true
	s.config.UpstreamRetryAttempts = 1
	s.userTaskGate = newUserTaskGate()
	sub := dao.LibraryReminderSubscription{StudentID: "20260001", Enabled: true, PreferenceVersion: 1, AuthStatus: dao.SubscriptionAuthOK}
	if err := db.Create(&sub).Error; err != nil {
		t.Fatal(err)
	}
	s.crawler = reminderTestCrawler{teamErr: errors.New("upstream failure")}
	if err := s.scanTeamUser(ctx, sub); err == nil {
		t.Fatal("expected failure")
	}
	var stored dao.LibraryReminderSubscription
	db.First(&stored, sub.ID)
	if stored.TeamBaselineCompleted || stored.LastTeamScanAt != nil || stored.LastTeamScanAttemptAt == nil || stored.AuthStatus != dao.SubscriptionAuthOK {
		t.Fatalf("failed scan modified state: %+v", stored)
	}
}

func TestTeamSuccessWithoutBaselineAndPreferenceReset(t *testing.T) {
	ctx := context.Background()
	s, db := newReminderTestService(t)
	s.config.NotificationTypes.TeamSuccess = true
	s.userTaskGate = newUserTaskGate()
	sub := dao.LibraryReminderSubscription{StudentID: "20260001", Enabled: true, FeedRevision: 1, PreferenceVersion: 1}
	if err := db.Create(&sub).Error; err != nil {
		t.Fatal(err)
	}
	s.crawler = reminderTestCrawler{team: &crawler.ReminderTeam{ID: "team-1", Status: 1}}
	if err := s.scanTeamUser(ctx, sub); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&dao.NotificationOutbox{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("outbox=%d err=%v", count, err)
	}
	_, err := s.dao.ApplyPreferenceChanges(ctx, []dao.PreferenceChange{{Revision: 2, StudentID: sub.StudentID, Enabled: false}, {Revision: 3, StudentID: sub.StudentID, Enabled: true}}, 3)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := s.dao.Subscription(ctx, sub.StudentID)
	if err != nil || updated.TeamBaselineCompleted || updated.LastTeamScanAt != nil || updated.LastTeamScanAttemptAt != nil {
		t.Fatalf("reset sub=%+v err=%v", updated, err)
	}
	if err := s.scanTeamUser(ctx, *updated); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&dao.NotificationOutbox{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("duplicate outbox=%d err=%v", count, err)
	}
}

func TestTeamSubscriptionsPagesOldestAttemptsFirst(t *testing.T) {
	ctx := context.Background()
	s, db := newReminderTestService(t)
	now := s.now()
	for i := 0; i < 205; i++ {
		row := dao.LibraryReminderSubscription{StudentID: fmt.Sprintf("%08d", i), Enabled: true, PreferenceVersion: 1}
		if i < 3 {
			at := now.Add(-5 * time.Minute)
			row.LastTeamScanAttemptAt = &at
		}
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	cutoff := now.Add(-s.config.TeamScanMinInterval)
	var after *time.Time
	var id int64
	seen := make(map[int64]bool)
	for {
		rows, err := s.dao.TeamSubscriptions(ctx, cutoff, after, id, 20)
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) == 0 {
			break
		}
		if len(seen) == 0 && rows[0].LastTeamScanAttemptAt != nil {
			t.Fatal("未尝试用户未被优先选择")
		}
		for _, row := range rows {
			if seen[row.ID] {
				t.Fatalf("重复扫描订阅 %d", row.ID)
			}
			seen[row.ID] = true
			if _, err := s.dao.MarkTeamAttempt(ctx, row.StudentID, row.PreferenceVersion, now); err != nil {
				t.Fatal(err)
			}
		}
		last := rows[len(rows)-1]
		id, after = last.ID, last.LastTeamScanAttemptAt
	}
	if len(seen) != 205 {
		t.Fatalf("扫描数量=%d", len(seen))
	}
}

type teamScanTestCrawler struct {
	reminderTestCrawler
	getTeam func(context.Context) (*crawler.ReminderTeam, error)
}

func (c teamScanTestCrawler) GetCurrentTeam(ctx context.Context, _ string) (*crawler.ReminderTeam, error) {
	return c.getTeam(ctx)
}

func TestScanTeamsAggregatesPages(t *testing.T) {
	for _, tc := range []struct {
		name   string
		fail   bool
		cancel bool
	}{
		{name: "all successful"},
		{name: "multiple failed pages", fail: true},
		{name: "canceled after failed page", fail: true, cancel: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, db := newReminderTestService(t)
			s.config.NotificationTypes.TeamSuccess = true
			s.config.UserConcurrency = 1
			s.config.UserJitter = 0
			s.config.UpstreamRetryAttempts = 1
			s.userTaskGate = newUserTaskGate()
			rows := make([]dao.LibraryReminderSubscription, 2*teamScanPageSize+5)
			for i := range rows {
				rows[i] = dao.LibraryReminderSubscription{StudentID: fmt.Sprintf("2026%04d", i), Enabled: true, PreferenceVersion: 1}
			}
			if err := db.CreateInBatches(&rows, 100).Error; err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			calls := 0
			s.crawler = teamScanTestCrawler{getTeam: func(context.Context) (*crawler.ReminderTeam, error) {
				calls++
				if !tc.fail || calls <= teamScanPageSize {
					return nil, nil
				}
				if calls <= 2*teamScanPageSize {
					return nil, crawler.ErrUpstreamStateUnknown
				}
				if tc.cancel {
					cancel()
					return nil, context.Canceled
				}
				return nil, context.DeadlineExceeded
			}}
			err := s.ScanTeams(ctx)
			if !tc.fail {
				if err != nil || calls != len(rows) {
					t.Fatalf("calls=%d err=%v", calls, err)
				}
				return
			}
			var batch *subscriptionBatchError
			if !errors.As(err, &batch) {
				t.Fatalf("expected batch error, got %v", err)
			}
			if batch.Total != len(rows) || batch.Succeeded != teamScanPageSize || batch.Groups["invalid_response"] != teamScanPageSize {
				t.Fatalf("batch=%+v groups=%v", batch, batch.Groups)
			}
			if batch.Launched != batch.Succeeded+batch.Failed || batch.Total != batch.Launched+batch.Canceled || len(batch.Samples) != batchErrorSampleLimit {
				t.Fatalf("batch=%+v samples=%d", batch, len(batch.Samples))
			}
			if tc.cancel {
				if !errors.Is(err, context.Canceled) || batch.Groups["context_canceled"] == 0 || batch.Failed != teamScanPageSize+batch.Groups["context_canceled"] {
					t.Fatalf("cancellation lost: batch=%+v groups=%v cause=%v", batch, batch.Groups, batch.Cause)
				}
			} else if batch.Failed != teamScanPageSize+5 || batch.Canceled != 0 || batch.Groups["upstream_timeout"] != 5 || !errors.Is(err, crawler.ErrUpstreamStateUnknown) {
				t.Fatalf("batch=%+v groups=%v cause=%v", batch, batch.Groups, batch.Cause)
			}
		})
	}
}

// 7/8 仅为合成测试状态，不代表学校真实枚举。
type invitationTestCrawler struct {
	crawler.ReminderCrawler
	team  *crawler.InvitationTeam
	err   error
	calls int
	mu    sync.Mutex
}

func (c *invitationTestCrawler) GetCurrentTeamForInvitation(context.Context, string) (*crawler.InvitationTeam, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	return c.team, c.err
}

func newInvitationTestService(t *testing.T) (*ReminderService, *gorm.DB, *invitationTestCrawler) {
	t.Helper()
	s, db := newReminderTestService(t)
	s.config.NotificationTypes.TeamInvitation = true
	s.config.TeamInvitation.ContractVerified = true
	s.config.TeamInvitation.PendingMemberStatuses = []int{7}
	s.config.TeamInvitation.KnownMemberStatuses = []int{1, 7, 8}
	s.invitation.preferenceCaughtUp.Store(s.now().UnixNano())
	expires := s.now().Add(time.Hour)
	c := &invitationTestCrawler{team: &crawler.InvitationTeam{ReminderTeam: crawler.ReminderTeam{ID: "123", Status: 0, ExpirationTime: &expires}, OperatorStudentID: "operator", IsMasterUser: true, Members: map[string]int{"A": 7, "B": 7, "C": 7}}}
	s.crawler = c
	for _, id := range []string{"A", "B"} {
		if err := db.Create(&dao.LibraryReminderSubscription{StudentID: id, Enabled: true, PreferenceVersion: 3}).Error; err != nil {
			t.Fatal(err)
		}
	}
	return s, db, c
}

func assertInvitationCounts(t *testing.T, db *gorm.DB, facts, outbox int64) {
	t.Helper()
	for _, tc := range []struct {
		model any
		want  int64
	}{{&dao.LibraryTeamInvitation{}, facts}, {&dao.NotificationOutbox{}, outbox}, {&dao.LibraryTeamSnapshot{}, 0}, {&dao.NotificationJob{}, 0}} {
		var count int64
		if err := db.Model(tc.model).Count(&count).Error; err != nil || count != tc.want {
			t.Fatalf("model=%T count=%d want=%d err=%v", tc.model, count, tc.want, err)
		}
	}
}

func TestInvitationIdempotencyAndLocalPreferenceSuppression(t *testing.T) {
	s, db, c := newInvitationTestService(t)
	ctx := context.Background()
	if err := s.NotifyTeamInvitation(ctx, "operator", "123", []string{"B", "A", "A", "C"}); err != nil {
		t.Fatal(err)
	}
	assertInvitationCounts(t, db, 3, 2)
	var first dao.LibraryTeamInvitation
	if err := db.Where("recipient_student_id = ?", "A").First(&first).Error; err != nil {
		t.Fatal(err)
	}
	if !first.ExpiresAt.Equal(s.now().Add(15 * time.Minute)) {
		t.Fatalf("expires=%v", first.ExpiresAt)
	}
	var suppressed dao.LibraryTeamInvitation
	if err := db.Where("recipient_student_id = ?", "C").First(&suppressed).Error; err != nil {
		t.Fatal(err)
	}
	if suppressed.SuppressedReason != "subscription_disabled" {
		t.Fatalf("row=%+v", suppressed)
	}
	// 学校状态已变化、同步滞后甚至上游不可用，已存在的合法重试仍受理。
	c.err = errors.New("school unavailable")
	s.invitation = newInvitationState(4)
	if err := db.Create(&dao.LibraryReminderSubscription{StudentID: "C", Enabled: true, PreferenceVersion: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.NotifyTeamInvitation(ctx, "operator", "123", []string{"C", "B", "A"}); err != nil {
		t.Fatal(err)
	}
	if c.calls != 1 {
		t.Fatalf("duplicate queried school: %d", c.calls)
	}
	if err := s.NotifyTeamInvitation(ctx, "another", "123", []string{"A"}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("operator mismatch: %v", err)
	}
	// 接收者版本而非队长版本；发布不进入座位快照复核。
	rows, err := s.dao.ClaimOutbox(ctx, s.now(), 10, 10)
	if err != nil || len(rows) != 2 {
		t.Fatalf("rows=%v err=%v", rows, err)
	}
	for _, row := range rows {
		if row.PreferenceVersion != 3 || row.ExternalReservationID != "" {
			t.Fatalf("row=%+v", row)
		}
		if err := s.sendOutboxRow(ctx, row); err != nil {
			t.Fatal(err)
		}
	}
	if s.feed.(*reminderTestFeed).published != 2 {
		t.Fatal("not published")
	}
	// 清理 Outbox 不清理邀请凭证，不因重新开启偏好或重复上报补发。
	if err := s.dao.CleanupHistory(ctx, time.Now().Add(time.Hour), 10); err != nil {
		t.Fatal(err)
	}
	if err := s.NotifyTeamInvitation(ctx, "operator", "123", []string{"A", "B", "C"}); err != nil {
		t.Fatal(err)
	}
	assertInvitationCounts(t, db, 3, 0)
}

func TestInvitationPartialOverlapAndDailyLimit(t *testing.T) {
	s, db, c := newInvitationTestService(t)
	s.config.TeamInvitation.RecipientDailyLimit = 1
	ctx := context.Background()
	if err := s.NotifyTeamInvitation(ctx, "operator", "123", []string{"A"}); err != nil {
		t.Fatal(err)
	}
	c.team.Members["A"] = 1
	if err := s.NotifyTeamInvitation(ctx, "operator", "123", []string{"B", "A"}); err != nil {
		t.Fatal(err)
	}
	c.team.ID = "124"
	c.team.Members["A"] = 7
	if err := s.NotifyTeamInvitation(ctx, "operator", "124", []string{"A", "B"}); err != nil {
		t.Fatal(err)
	}
	assertInvitationCounts(t, db, 4, 2)
	var limited int64
	if err := db.Model(&dao.LibraryTeamInvitation{}).Where("suppressed_reason = ?", "recipient_limit").Count(&limited).Error; err != nil || limited != 2 {
		t.Fatalf("limited=%d err=%v", limited, err)
	}
	// 中国时区自然日切换后新队伍可通知，旧事实不复活。
	next := s.now().Add(24 * time.Hour)
	s.now = func() time.Time { return next }
	s.invitation.preferenceCaughtUp.Store(next.UnixNano())
	expires := next.Add(time.Hour)
	c.team.ID = "125"
	c.team.ExpirationTime = &expires
	if err := s.NotifyTeamInvitation(ctx, "operator", "125", []string{"A"}); err != nil {
		t.Fatal(err)
	}
	assertInvitationCounts(t, db, 5, 3)
}

func TestInvitationRejectsWithoutNewWrites(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*ReminderService, *invitationTestCrawler)
		want   codes.Code
	}{
		{"disabled", func(s *ReminderService, c *invitationTestCrawler) { s.config.NotificationTypes.TeamInvitation = false }, codes.Unavailable},
		{"dry_run", func(s *ReminderService, c *invitationTestCrawler) { yes := true; s.config.DryRun = &yes }, codes.Unavailable},
		{"unverified", func(s *ReminderService, c *invitationTestCrawler) { s.config.TeamInvitation.ContractVerified = false }, codes.Unavailable},
		{"not_ready", func(s *ReminderService, c *invitationTestCrawler) { s.invitation.preferenceCaughtUp.Store(0) }, codes.Unavailable},
		{"stale", func(s *ReminderService, c *invitationTestCrawler) {
			s.invitation.preferenceCaughtUp.Store(s.now().Add(-31 * time.Second).UnixNano())
		}, codes.Unavailable},
		{"school_failure", func(s *ReminderService, c *invitationTestCrawler) { c.err = errors.New("secret upstream body") }, codes.Unavailable},
		{"no_team", func(s *ReminderService, c *invitationTestCrawler) { c.team = nil }, codes.FailedPrecondition},
		{"different_team", func(s *ReminderService, c *invitationTestCrawler) { c.team.ID = "999" }, codes.FailedPrecondition},
		{"not_master", func(s *ReminderService, c *invitationTestCrawler) { c.team.IsMasterUser = false }, codes.PermissionDenied},
		{"different_account", func(s *ReminderService, c *invitationTestCrawler) { c.team.OperatorStudentID = "other" }, codes.PermissionDenied},
		{"success_team", func(s *ReminderService, c *invitationTestCrawler) { c.team.Status = 1 }, codes.FailedPrecondition},
		{"expired", func(s *ReminderService, c *invitationTestCrawler) { now := s.now(); c.team.ExpirationTime = &now }, codes.FailedPrecondition},
		{"unknown_member", func(s *ReminderService, c *invitationTestCrawler) { c.team.Members["C"] = 99 }, codes.Unavailable},
		{"accepted", func(s *ReminderService, c *invitationTestCrawler) { c.team.Members["B"] = 1 }, codes.FailedPrecondition},
		{"rejected", func(s *ReminderService, c *invitationTestCrawler) { c.team.Members["B"] = 8 }, codes.FailedPrecondition},
		{"missing_member", func(s *ReminderService, c *invitationTestCrawler) { delete(c.team.Members, "B") }, codes.FailedPrecondition},
		{"configured_limit", func(s *ReminderService, c *invitationTestCrawler) { s.config.TeamInvitation.MaxRecipients = 1 }, codes.InvalidArgument},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, db, c := newInvitationTestService(t)
			tc.change(s, c)
			err := s.NotifyTeamInvitation(context.Background(), "operator", "123", []string{"A", "B"})
			if status.Code(err) != tc.want {
				t.Fatalf("err=%v want=%v", err, tc.want)
			}
			assertInvitationCounts(t, db, 0, 0)
		})
	}
}

func TestInvitationBatchRollbackAndConcurrency(t *testing.T) {
	s, db, _ := newInvitationTestService(t)
	ctx := context.Background()
	// 第二位收件人的写入失败，第一位的邀请和 Outbox 也必须回滚。
	if err := db.Exec(`CREATE TRIGGER fail_second BEFORE INSERT ON notification_outbox WHEN NEW.student_id = 'B' BEGIN SELECT RAISE(ABORT, 'injected failure'); END`).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.NotifyTeamInvitation(ctx, "operator", "123", []string{"B", "A"}); status.Code(err) != codes.Internal {
		t.Fatalf("err=%v", err)
	}
	assertInvitationCounts(t, db, 0, 0)
	if err := db.Exec("DROP TRIGGER fail_second").Error; err != nil {
		t.Fatal(err)
	}
	// SQLite 不支持 FOR UPDATE，以单连接执行事务验证并发重试收敛；生产锁需 MySQL 联调。
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	var wg sync.WaitGroup
	results := make(chan error, 4)
	for range 4 {
		wg.Add(1)
		go func() { defer wg.Done(); results <- s.NotifyTeamInvitation(ctx, "operator", "123", []string{"B", "A"}) }()
	}
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	assertInvitationCounts(t, db, 2, 2)
}

func TestInvitationFrequencyAndExpiry(t *testing.T) {
	s, db, c := newInvitationTestService(t)
	ctx := context.Background()
	s.config.TeamInvitation.RequestsPerMinute = 2
	c.err = errors.New("temporary")
	for range 2 {
		if err := s.NotifyTeamInvitation(ctx, "operator", "123", []string{"A"}); status.Code(err) != codes.Unavailable {
			t.Fatal(err)
		}
	}
	if err := s.NotifyTeamInvitation(ctx, "operator", "123", []string{"A"}); status.Code(err) != codes.ResourceExhausted {
		t.Fatal(err)
	}
	c.err = nil
	next := s.now().Add(time.Minute)
	s.now = func() time.Time { return next }
	s.invitation.preferenceCaughtUp.Store(next.UnixNano())
	expiry := next.Add(30 * time.Second)
	c.team.ExpirationTime = &expiry
	if err := s.NotifyTeamInvitation(ctx, "operator", "123", []string{"A"}); err != nil {
		t.Fatal(err)
	}
	rows, err := s.dao.ClaimOutbox(ctx, next, 10, 10)
	if err != nil || len(rows) != 1 {
		t.Fatalf("rows=%v err=%v", rows, err)
	}
	if !rows[0].ExpiresAt.Equal(expiry) {
		t.Fatalf("expiry=%v", rows[0].ExpiresAt)
	}
	next = expiry
	if err := s.sendOutboxRow(ctx, rows[0]); err != nil {
		t.Fatal(err)
	}
	var row dao.NotificationOutbox
	if err := db.First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Status != dao.OutboxSuppressed || s.feed.(*reminderTestFeed).published != 0 {
		t.Fatalf("row=%+v", row)
	}
}

func TestInvitationPayloadValidation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*notificationPayload, *dao.NotificationOutbox)
	}{
		{"valid", func(p *notificationPayload, o *dao.NotificationOutbox) {}},
		{"type", func(p *notificationPayload, o *dao.NotificationOutbox) { p.NotificationType = NotificationTeamSuccess }},
		{"no_team", func(p *notificationPayload, o *dao.NotificationOutbox) { p.TeamID = "" }},
		{"no_observation", func(p *notificationPayload, o *dao.NotificationOutbox) { p.TargetAt = 0 }},
		{"no_expiry", func(p *notificationPayload, o *dao.NotificationOutbox) { o.ExpiresAt = nil }},
		{"expiry_mismatch", func(p *notificationPayload, o *dao.NotificationOutbox) { p.InvitationExpiresAt++ }},
		{"reservation", func(p *notificationPayload, o *dao.NotificationOutbox) { o.ExternalReservationID = "seat" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, db, _ := newInvitationTestService(t)
			if err := s.NotifyTeamInvitation(context.Background(), "operator", "123", []string{"A"}); err != nil {
				t.Fatal(err)
			}
			rows, err := s.dao.ClaimOutbox(context.Background(), s.now(), 10, 10)
			if err != nil || len(rows) != 1 {
				t.Fatal(err)
			}
			row := rows[0]
			var payload notificationPayload
			if err := json.Unmarshal(row.Payload, &payload); err != nil {
				t.Fatal(err)
			}
			tc.mutate(&payload, &row)
			row.Payload, _ = json.Marshal(payload)
			if err := s.sendOutboxRow(context.Background(), row); err != nil {
				t.Fatal(err)
			}
			var stored dao.NotificationOutbox
			db.First(&stored)
			want := dao.OutboxSuppressed
			if tc.name == "valid" {
				want = dao.OutboxSent
			}
			if stored.Status != want {
				t.Fatalf("status=%s", stored.Status)
			}
			event := payloadFeedEvent(row.DedupeKey, payload)
			if event.ExtendFields["target_at"] != "" || event.Url != "" {
				t.Fatalf("event=%v", event)
			}
		})
	}
	if invitationDedupeKey("a:b", "12") == invitationDedupeKey("a", "12") {
		t.Fatal("ambiguous dedupe key")
	}
}

func (f *reminderTestFeed) ReminderUsers(ctx context.Context, after, revision int64, limit int32) ([]LibraryReminderUser, int64, int64, error) {
	return f.users(ctx, after, revision, limit)
}

func TestInvitationPreferenceReadinessRequiresCurrentProcessCatchup(t *testing.T) {
	s, _, _ := newInvitationTestService(t)
	baseline := true
	s.config.BaselineOnEnable = &baseline
	ctx := context.Background()
	if _, err := s.dao.ApplyPreferenceChanges(ctx, nil, 9); err != nil {
		t.Fatal(err)
	}
	s.invitation.preferenceCaughtUp.Store(0)
	if s.invitationPreferencesReady() {
		t.Fatal("persisted cursor is not readiness")
	}
	fullCalls := 0
	f := &reminderTestFeed{
		users: func(_ context.Context, after, revision int64, _ int32) ([]LibraryReminderUser, int64, int64, error) {
			fullCalls++
			return nil, 0, 9, nil
		},
		changes: func(_ context.Context, after int64, _ int32) ([]LibraryPreferenceChange, int64, error) {
			return nil, after, nil
		},
	}
	s.feed = f
	if err := s.syncPreferences(ctx); err != nil {
		t.Fatal(err)
	}
	if !s.invitationPreferencesReady() || fullCalls != 1 {
		t.Fatalf("ready=%v full=%d", s.invitationPreferencesReady(), fullCalls)
	}
	now := s.now().Add(31 * time.Second)
	s.now = func() time.Time { return now }
	if s.invitationPreferencesReady() {
		t.Fatal("stale sync accepted")
	}
	if err := s.syncPreferences(ctx); err != nil {
		t.Fatal(err)
	}
	if !s.invitationPreferencesReady() || fullCalls != 1 {
		t.Fatal("empty incremental page did not refresh readiness")
	}
	now = now.Add(31 * time.Second)
	f.changes = func(context.Context, int64, int32) ([]LibraryPreferenceChange, int64, error) {
		return nil, 0, errors.New("offline")
	}
	if err := s.syncPreferences(ctx); err == nil {
		t.Fatal("expected sync failure")
	}
	if s.invitationPreferencesReady() {
		t.Fatal("failed sync refreshed readiness")
	}
}

func TestInvitationConcurrencyAndLimiterCapacity(t *testing.T) {
	s, db, _ := newInvitationTestService(t)
	for range cap(s.invitation.inFlight) {
		s.invitation.inFlight <- struct{}{}
	}
	if err := s.NotifyTeamInvitation(context.Background(), "operator", "123", []string{"A"}); status.Code(err) != codes.ResourceExhausted {
		t.Fatal(err)
	}
	assertInvitationCounts(t, db, 0, 0)
	state := newInvitationState(1)
	for i := range invitationLimiterCapacity {
		state.windows[fmt.Sprint(i)] = invitationRequestWindow{started: s.now(), count: 1}
	}
	if state.allow("new", s.now(), 10) {
		t.Fatal("unbounded limiter")
	}
	if !state.allow("new", s.now().Add(time.Minute), 10) || len(state.windows) != 1 {
		t.Fatal("expired windows not cleaned")
	}
}
