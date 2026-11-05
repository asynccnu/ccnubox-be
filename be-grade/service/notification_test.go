package service

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/asynccnu/ccnubox-be/be-grade/domain"
	"github.com/asynccnu/ccnubox-be/be-grade/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-grade/repository/model"
	feedv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feed/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger/zapx"
	"github.com/asynccnu/ccnubox-be/common/pkg/testx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type notificationFeedClient struct {
	feedv1.FeedServiceClient
	publish func(context.Context, *feedv1.PublicFeedEventReq) (*feedv1.PublicFeedEventResp, error)
}

func (c notificationFeedClient) PublicFeedEvent(ctx context.Context, req *feedv1.PublicFeedEventReq, _ ...grpc.CallOption) (*feedv1.PublicFeedEventResp, error) {
	return c.publish(ctx, req)
}

func newGradeServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	if db := testx.MySQL(t); db != nil {
		return db
	}
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "notification.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

func newNotificationTestDB(t *testing.T) (*gorm.DB, model.GradeNotificationOutbox) {
	t.Helper()
	db := newGradeServiceTestDB(t)
	if err := db.AutoMigrate(&model.GradeNotificationOutbox{}); err != nil {
		t.Fatal(err)
	}
	id, _ := model.NewNotificationID()
	row, err := domain.NewGradeNotification(model.Grade{StudentId: "student", NotificationID: &id, ScoreVersion: 2,
		Kcmc: "旧学期课程", Kcxzmc: "含引号的\"课程性质", Xnm: 2020, Xqm: 1}, "score_changed", time.Now().Unix())
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	return db, row
}

func notificationRow(t *testing.T, db *gorm.DB, id int64) model.GradeNotificationOutbox {
	t.Helper()
	var row model.GradeNotificationOutbox
	if err := db.First(&row, id).Error; err != nil {
		t.Fatal(err)
	}
	return row
}

type failPublishedOnce struct {
	dao.NotificationDAO
	fail bool
}

func (d *failPublishedOnce) MarkPublished(ctx context.Context, id int64) error {
	if d.fail {
		d.fail = false
		return errors.New("write unavailable after downstream accepted")
	}
	return d.NotificationDAO.MarkPublished(ctx, id)
}

func TestNotificationPublisherRetriesSameSnapshotAfterRestartAndWriteFailure(t *testing.T) {
	db, row := newNotificationTestDB(t)
	var first *feedv1.FeedEvent
	calls := 0
	feed := notificationFeedClient{publish: func(ctx context.Context, req *feedv1.PublicFeedEventReq) (*feedv1.PublicFeedEventResp, error) {
		calls++
		if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > 3*time.Second || req.IsAll || req.StudentId != row.StudentID {
			t.Fatal("invalid recipient or unbounded RPC")
		}
		if calls == 1 {
			first = proto.Clone(req.Event).(*feedv1.FeedEvent)
			return &feedv1.PublicFeedEventResp{Status: feedv1.PublishStatus_ACCEPTED}, nil
		}
		if !proto.Equal(first, req.Event) || req.Event.DedupeKey != row.DedupeKey {
			t.Fatal("retry rebuilt event")
		}
		return &feedv1.PublicFeedEventResp{Status: feedv1.PublishStatus_DUPLICATE}, nil
	}}
	l := zapx.NewZapLogger(zap.NewNop())
	p := NewNotificationPublisher(&failPublishedOnce{NotificationDAO: dao.NewNotificationDAO(db), fail: true}, feed, l, nil)
	p.publishDue(context.Background())
	if got := notificationRow(t, db, row.ID); got.Status != model.NotificationPending || got.Payload != row.Payload || got.DedupeKey != row.DedupeKey {
		t.Fatal(got)
	}
	// 重建发布器模拟进程重启，不依赖内存任务或重新生成业务 key。
	NewNotificationPublisher(dao.NewNotificationDAO(db), feed, l, nil).publishDue(context.Background())
	if got := notificationRow(t, db, row.ID); got.Status != model.NotificationPublished || got.Payload != row.Payload || calls != 2 {
		t.Fatal(got, calls)
	}
}

func TestNotificationPublisherBackoffAndInvalidRecovery(t *testing.T) {
	for _, name := range []string{"timeout", "rpc validation", "broken payload", "unknown status"} {
		t.Run(name, func(t *testing.T) {
			db, row := newNotificationTestDB(t)
			if name == "broken payload" {
				if err := db.Model(&model.GradeNotificationOutbox{}).Where("id = ?", row.ID).Update("payload", "{").Error; err != nil {
					t.Fatal(err)
				}
			}
			calls := 0
			feed := notificationFeedClient{publish: func(context.Context, *feedv1.PublicFeedEventReq) (*feedv1.PublicFeedEventResp, error) {
				calls++
				switch name {
				case "timeout":
					return nil, status.Error(codes.DeadlineExceeded, "timeout")
				case "rpc validation":
					return nil, status.Error(codes.InvalidArgument, "invalid")
				default:
					return &feedv1.PublicFeedEventResp{Status: feedv1.PublishStatus_SUPPRESSED_BY_ALLOW_LIST}, nil
				}
			}}
			d := dao.NewNotificationDAO(db)
			p := NewNotificationPublisher(d, feed, zapx.NewZapLogger(zap.NewNop()), nil)
			p.publishDue(context.Background())
			got := notificationRow(t, db, row.ID)
			invalid := name == "rpc validation" || name == "broken payload"
			wantStatus := model.NotificationPending
			if invalid {
				wantStatus = model.NotificationInvalid
			}
			if got.Status != wantStatus || got.Attempts != 1 || got.LastError == "" || got.NextAttemptAt < time.Now().Unix()+4 {
				t.Fatal(got)
			}
			if name == "broken payload" && calls != 0 {
				t.Fatal("published broken event")
			}
			stats, err := d.Stats(context.Background())
			if err != nil || stats.Pending+stats.Invalid != 1 || (invalid && stats.Invalid != 1) || (!invalid && stats.Oldest != row.CreatedAt) {
				t.Fatal(stats, err)
			}
			if invalid {
				if name == "broken payload" {
					if err := dao.RequeueInvalidGradeNotification(context.Background(), db, row.ID); err == nil {
						t.Fatal("broken payload requeued without validation")
					}
					if err := db.Model(&got).Update("payload", row.Payload).Error; err != nil {
						t.Fatal(err)
					}
				}
				if err := dao.RequeueInvalidGradeNotification(context.Background(), db, row.ID); err != nil {
					t.Fatal(err)
				}
				if got = notificationRow(t, db, row.ID); got.Status != model.NotificationPending || got.DedupeKey != row.DedupeKey || got.Attempts != 1 {
					t.Fatal(got)
				}
			}
		})
	}
}

func TestNotificationPublisherCancelsInflightRPC(t *testing.T) {
	db, row := newNotificationTestDB(t)
	entered := make(chan struct{})
	feed := notificationFeedClient{publish: func(ctx context.Context, _ *feedv1.PublicFeedEventReq) (*feedv1.PublicFeedEventResp, error) {
		close(entered)
		<-ctx.Done()
		return nil, ctx.Err()
	}}
	p := NewNotificationPublisher(dao.NewNotificationDAO(db), feed, zapx.NewZapLogger(zap.NewNop()), nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { defer close(done); p.Run(ctx) }()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("publisher did not enter RPC")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("publisher did not stop")
	}
	if got := notificationRow(t, db, row.ID); got.Status != model.NotificationPending || got.Attempts != 0 {
		t.Fatal(got)
	}
}

func TestGradeNotificationRejectsUnactivatedIdentity(t *testing.T) {
	id, _ := model.NewNotificationID()
	empty := ""
	for _, grade := range []model.Grade{
		{StudentId: "student"},
		{StudentId: "student", ScoreVersion: 1},
		{StudentId: "student", NotificationID: &id},
		{StudentId: "student", NotificationID: &empty, ScoreVersion: 1},
	} {
		if _, err := domain.NewGradeNotification(grade, "score_changed", time.Now().Unix()); err == nil {
			t.Fatal("unactivated identity accepted", grade)
		}
	}
	row, err := domain.NewGradeNotification(model.Grade{StudentId: "student", NotificationID: &id, ScoreVersion: 1}, "score_changed", time.Now().Unix())
	if err != nil {
		t.Fatal("first tracked version may be a score change", err)
	}
	if event, err := domain.ParseGradeNotification(row); err != nil || event.ExtendFields["change_reason"] != "score_changed" {
		t.Fatal(event, err)
	}
}

func TestGradeNotificationKeyOnlyUsesBusinessIdentity(t *testing.T) {
	if got := domain.GradeNotificationKey("student", "identity", 1); got != "grade:v2:b767a0ea5b76eef6d67fafc4c6b2fe8e68a9a0e0677368dce38e41a24cd3200c" {
		// 独立固定向量约束 JSON 数组的字段顺序、字符串和整数编码。
		t.Fatal(got)
	}
	if domain.GradeNotificationKey("ab", "c", 1) == domain.GradeNotificationKey("a", "bc", 1) ||
		domain.GradeNotificationKey("s", "id", 2) == domain.GradeNotificationKey("s", "id", 3) {
		t.Fatal("different business identity shared a key")
	}
	for _, tc := range []struct {
		attempts int64
		want     time.Duration
	}{{1, 5 * time.Second}, {2, 10 * time.Second}, {7, 5 * time.Minute}, {99999, 5 * time.Minute}} {
		if got := notificationBackoff(tc.attempts); got != tc.want {
			t.Fatal(tc, got)
		}
	}
}
