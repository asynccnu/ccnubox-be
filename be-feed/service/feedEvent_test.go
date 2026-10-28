package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	"github.com/asynccnu/ccnubox-be/be-feed/events/producer"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/model"
	feedv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feed/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger/zapx"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type broadcastProducer struct {
	producer.Producer
	calls  []domain.FeedEvent
	failAt int
}

func (p *broadcastProducer) SendMessage(_ context.Context, _ string, event domain.FeedEvent) error {
	p.calls = append(p.calls, event)
	if len(p.calls) == p.failAt {
		return errors.New("Kafka unavailable")
	}
	return nil
}

type broadcastUsers struct{ dao.FeedUserConfigDAO }

func (broadcastUsers) GetStudentIdsByCursor(_ context.Context, lastID int64, _ int) ([]string, int64, error) {
	if lastID == 0 {
		prefix := fmt.Sprintf("%d", time.Now().Year()-1)
		return []string{prefix + "0001", prefix + "0002", prefix + "0003"}, 3, nil
	}
	return nil, 3, nil
}

type broadcastEvents struct {
	dao.FeedEventDAO
	stored map[string]bool
	err    error
}

func (d broadcastEvents) DedupeKeyExists(_ context.Context, studentID, key string) (bool, error) {
	return d.stored[studentID+":"+key], d.err
}

func (d broadcastEvents) DedupeKeyExistsBatch(_ context.Context, studentIDs []string, key string) (map[string]bool, error) {
	if d.err != nil {
		return nil, d.err
	}
	existing := make(map[string]bool, len(studentIDs))
	for _, studentID := range studentIDs {
		if d.stored[studentID+":"+key] {
			existing[studentID] = true
		}
	}
	return existing, nil
}

type libraryPublishUsers struct {
	dao.FeedUserConfigDAO
	enabled bool
}

func (d libraryPublishUsers) IsLibraryEnabled(context.Context, string) (bool, error) {
	return d.enabled, nil
}

func TestSinglePublishDedupeByRecipientAndKey(t *testing.T) {
	for _, eventType := range []string{"muxi", "grade", "holiday", "energy", "feedback", "library"} {
		t.Run(eventType, func(t *testing.T) {
			for _, tc := range []struct {
				name      string
				studentID string
				key       string
				duplicate bool
			}{
				{name: "stored message", studentID: "a", key: "announcement", duplicate: true},
				{name: "different recipient", studentID: "b", key: "announcement"},
				{name: "different key", studentID: "a", key: "another"},
				{name: "generated key", studentID: "a"},
			} {
				t.Run(tc.name, func(t *testing.T) {
					p := &broadcastProducer{}
					s := &feedEventService{feedProducer: p,
						feedEventDAO:      broadcastEvents{stored: map[string]bool{"a:announcement": true}},
						feedUserConfigDAO: libraryPublishUsers{enabled: true}}
					event := domain.FeedEvent{StudentId: tc.studentID, Type: eventType, DedupeKey: tc.key}
					status, err := s.PublicFeedEvent(context.Background(), false, event)
					if err != nil {
						t.Fatal(err)
					}
					if tc.duplicate {
						if status != feedv1.PublishStatus_DUPLICATE || len(p.calls) != 0 {
							t.Fatalf("duplicate was sent: status=%v calls=%+v", status, p.calls)
						}
						return
					}
					if status != feedv1.PublishStatus_ACCEPTED || len(p.calls) != 1 {
						t.Fatalf("new message was not sent: status=%v calls=%+v", status, p.calls)
					}
					sent := p.calls[0]
					if sent.StudentId != tc.studentID || sent.DedupeKey == "" || (tc.key != "" && sent.DedupeKey != tc.key) {
						t.Fatalf("unexpected event: %+v", sent)
					}
				})
			}
		})
	}
}

func TestPublishStopsOnDedupeLookupFailure(t *testing.T) {
	for _, broadcast := range []bool{false, true} {
		t.Run(fmt.Sprintf("broadcast=%t", broadcast), func(t *testing.T) {
			p := &broadcastProducer{}
			s := &feedEventService{feedProducer: p, feedUserConfigDAO: broadcastUsers{},
				feedEventDAO: broadcastEvents{err: errors.New("database unavailable")}}
			event := domain.FeedEvent{StudentId: "a", Type: "muxi", DedupeKey: "announcement"}
			if _, err := s.PublicFeedEvent(context.Background(), broadcast, event); err == nil {
				t.Fatal("dedupe lookup failure was ignored")
			}
			if len(p.calls) != 0 {
				t.Fatalf("sent without dedupe check: %+v", p.calls)
			}
		})
	}
}

func TestLibraryPublishStillHonorsPreferenceBeforeDedupe(t *testing.T) {
	p := &broadcastProducer{}
	s := &feedEventService{feedProducer: p, feedUserConfigDAO: libraryPublishUsers{enabled: false}}
	status, err := s.PublicFeedEvent(context.Background(), false, domain.FeedEvent{StudentId: "a", Type: "library", DedupeKey: "reminder"})
	if err != nil || status != feedv1.PublishStatus_SUPPRESSED_BY_ALLOW_LIST || len(p.calls) != 0 {
		t.Fatalf("status=%v err=%v calls=%+v", status, err, p.calls)
	}
}

func TestTargetedPublishRetrySkipsStoredRecipients(t *testing.T) {
	p := &broadcastProducer{failAt: 2}
	stored := map[string]bool{}
	s := &feedEventService{feedProducer: p, feedEventDAO: broadcastEvents{stored: stored},
		l: zapx.NewZapLogger(zap.NewNop())}
	event := domain.FeedEvent{Type: "muxi", DedupeKey: "muxi:targeted"}
	publish := func() error {
		for _, studentID := range []string{"a", "b", "c"} {
			event.StudentId = studentID
			if _, err := s.PublicFeedEvent(context.Background(), false, event); err != nil {
				return err
			}
		}
		return nil
	}
	if err := publish(); err == nil {
		t.Fatal("targeted publish hid partial failure")
	}
	if len(p.calls) != 2 || p.calls[1].StudentId != "b" {
		t.Fatalf("continued after failure: %+v", p.calls)
	}
	// Kafka 消费者已将第一位收件人的消息落库，再用原 key 重跑整个白名单。
	stored["a:"+event.DedupeKey] = true
	p.calls = nil
	p.failAt = 0
	if err := publish(); err != nil {
		t.Fatal(err)
	}
	if len(p.calls) != 2 || p.calls[0].StudentId != "b" || p.calls[1].StudentId != "c" {
		t.Fatalf("resent stored recipient: %+v", p.calls)
	}
	for _, sent := range p.calls {
		if sent.DedupeKey != event.DedupeKey {
			t.Fatalf("retry changed dedupe key: %+v", sent)
		}
	}
}

func TestBroadcastStopsOnFailureAndReusesDedupeKey(t *testing.T) {
	p := &broadcastProducer{failAt: 2}
	stored := map[string]bool{}
	s := &feedEventService{feedProducer: p, feedUserConfigDAO: broadcastUsers{}, feedEventDAO: broadcastEvents{stored: stored},
		l: zapx.NewZapLogger(zap.NewNop())}
	event := domain.FeedEvent{Type: "muxi", DedupeKey: "announcement"}
	if _, err := s.PublicFeedEvent(context.Background(), true, event); err == nil {
		t.Fatal("broadcast hid partial failure")
	}
	if len(p.calls) != 2 || p.calls[1].StudentId != fmt.Sprintf("%d0002", time.Now().Year()-1) {
		t.Fatalf("continued after failure: %+v", p.calls)
	}
	stored[fmt.Sprintf("%d0001:announcement", time.Now().Year()-1)] = true
	p.calls = nil
	p.failAt = 0
	if _, err := s.PublicFeedEvent(context.Background(), true, event); err != nil {
		t.Fatal(err)
	}
	if len(p.calls) != 2 || p.calls[0].StudentId != fmt.Sprintf("%d0002", time.Now().Year()-1) || p.calls[1].StudentId != fmt.Sprintf("%d0003", time.Now().Year()-1) {
		t.Fatalf("resent completed prefix: %+v", p.calls)
	}
	for _, sent := range p.calls {
		if sent.DedupeKey != event.DedupeKey {
			t.Fatal("retry changed dedupe key")
		}
	}
}

func TestBroadcastRejectsMissingKeyBeforeSending(t *testing.T) {
	p := &broadcastProducer{}
	s := &feedEventService{feedProducer: p, l: zapx.NewZapLogger(zap.NewNop())}
	if _, err := s.PublicFeedEvent(context.Background(), true, domain.FeedEvent{Type: "muxi"}); err == nil {
		t.Fatal("broadcast accepted missing dedupe key")
	}
	if len(p.calls) != 0 {
		t.Fatal("sent invalid broadcast")
	}
}

func TestLegacyOfficialBroadcastSkipsGraduatedAndMalformedIDs(t *testing.T) {
	p := &broadcastProducer{}
	current := fmt.Sprintf("%d0001", time.Now().Year()-1)
	users := &officialUsersStub{batches: [][]string{{"19990001", "bad"}, {current, "20990001"}}}
	s := &feedEventService{feedProducer: p, feedUserConfigDAO: users, feedEventDAO: broadcastEvents{stored: map[string]bool{}}, l: zapx.NewZapLogger(zap.NewNop())}
	if _, err := s.PublicFeedEvent(context.Background(), true, domain.FeedEvent{Type: "muxi", DedupeKey: "legacy"}); err != nil {
		t.Fatal(err)
	}
	if len(p.calls) != 1 || p.calls[0].StudentId != current || len(users.cursors) != 3 {
		t.Fatalf("events=%+v cursors=%v", p.calls, users.cursors)
	}
}

func TestGetFeedEventsHidesHistoryAndLegacyFailuresUntilEnabled(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	if err := db.AutoMigrate(&model.FeedEvent{}, &model.FeedFailEvent{}, &model.FeedUserConfig{}, &model.FeedPushDelivery{}); err != nil {
		t.Fatal(err)
	}
	const sid = "20260001"
	config := model.FeedUserConfig{StudentId: sid, PushConfig: 0}
	if err := db.Create(&config).Error; err != nil {
		t.Fatal(err)
	}
	event := model.FeedEvent{StudentId: sid, Type: "GrAdE", DedupeKey: "old", ExtendFields: model.ExtendFields{}}
	failed := model.FeedFailEvent{StudentId: sid, Type: "GrAdE", ExtendFields: model.ExtendFields{}}
	if err := db.Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&failed).Error; err != nil {
		t.Fatal(err)
	}
	delivery := model.FeedPushDelivery{FeedEventID: event.ID, StudentId: sid, Status: model.PushDeliverySuppressed}
	if err := db.Create(&delivery).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewFeedEventService(dao.NewFeedEventDAO(db), nil, dao.NewFeedUserConfigDAO(db), dao.NewFeedFailEventDAO(db), nil, zapx.NewZapLogger(zap.NewNop()))
	ctx := context.Background()
	consumed := false
	for _, enabled := range []bool{false, true, false, true} {
		bits := uint16(0)
		if enabled {
			bits = 1 << model.GradePos
		}
		if err := db.Model(&config).Update("push_config", bits).Error; err != nil {
			t.Fatal(err)
		}
		events, fail, err := svc.GetFeedEvents(ctx, sid)
		if err != nil {
			t.Fatal(err)
		}
		if !enabled && (len(events) != 0 || len(fail) != 0) {
			t.Fatalf("disabled leaked events=%v fail=%v", events, fail)
		}
		if enabled && (len(events) != 1 || events[0].Type != "grade") {
			t.Fatalf("history not restored: %v", events)
		}
		wantFail := 0
		if enabled && !consumed {
			wantFail = 1
			consumed = true
		}
		if len(fail) != wantFail {
			t.Fatalf("legacy failures=%d, want=%d", len(fail), wantFail)
		}
		for _, event := range fail {
			if event.Type != "grade" {
				t.Fatalf("failure type=%s", event.Type)
			}
		}
	}
	var count int64
	if err := db.Unscoped().Model(&model.FeedFailEvent{}).Where("deleted_at IS NOT NULL").Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("legacy failure not consumed: %d %v", count, err)
	}
	if err := db.First(&delivery, delivery.ID).Error; err != nil || delivery.Status != model.PushDeliverySuppressed {
		t.Fatalf("cancelled delivery restored: %+v err=%v", delivery, err)
	}
}

type failEventsAfterReadDAO struct {
	dao.FeedFailEventDAO
	afterRead func() error
}

func (d *failEventsAfterReadDAO) GetFeedFailEventsByStudentId(ctx context.Context, studentId string) ([]model.FeedFailEvent, error) {
	events, err := d.FeedFailEventDAO.GetFeedFailEventsByStudentId(ctx, studentId)
	if err != nil {
		return nil, err
	}
	if d.afterRead != nil {
		if err := d.afterRead(); err != nil {
			return nil, err
		}
	}
	return events, nil
}

func TestGetFeedEventsOnlyConsumesReadFailures(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	if err := db.AutoMigrate(&model.FeedEvent{}, &model.FeedFailEvent{}, &model.FeedUserConfig{}, &model.FeedPushDelivery{}); err != nil {
		t.Fatal(err)
	}
	const sid = "20260001"
	config := model.FeedUserConfig{StudentId: sid, PushConfig: 1 << model.GradePos}
	if err := db.Create(&config).Error; err != nil {
		t.Fatal(err)
	}
	failures := []model.FeedFailEvent{
		{StudentId: sid, Type: "grade"},
		{StudentId: sid, Type: "muxi"},
		{StudentId: "20260002", Type: "grade"},
	}
	if err := db.Create(&failures).Error; err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	configDAO := dao.NewFeedUserConfigDAO(db)
	failDAO := &failEventsAfterReadDAO{FeedFailEventDAO: dao.NewFeedFailEventDAO(db)}
	newFailure := model.FeedFailEvent{StudentId: sid, Type: "grade"}
	// 固定交错顺序：读取 grade 后开启 muxi，并插入一条同类型新记录，再执行消费删除。
	failDAO.afterRead = func() error {
		if _, err := configDAO.ChangeConfigBits(ctx, sid, map[int]bool{model.MuxiPos: true}, nil); err != nil {
			return err
		}
		return db.Create(&newFailure).Error
	}
	svc := NewFeedEventService(dao.NewFeedEventDAO(db), nil, configDAO, failDAO, nil, zapx.NewZapLogger(zap.NewNop()))
	_, failed, err := svc.GetFeedEvents(ctx, sid)
	if err != nil || len(failed) != 1 || failed[0].ID != failures[0].ID {
		t.Fatalf("consumed failures=%v err=%v", failed, err)
	}
	var remaining []model.FeedFailEvent
	if err := db.Order("id ASC").Find(&remaining).Error; err != nil {
		t.Fatal(err)
	}
	wantIDs := []int64{failures[1].ID, failures[2].ID, newFailure.ID}
	if len(remaining) != len(wantIDs) {
		t.Fatalf("unread failures=%v wantIDs=%v", remaining, wantIDs)
	}
	for i := range wantIDs {
		if remaining[i].ID != wantIDs[i] {
			t.Fatalf("remaining[%d]=%d want=%d", i, remaining[i].ID, wantIDs[i])
		}
	}
	failDAO.afterRead = nil
	_, failed, err = svc.GetFeedEvents(ctx, sid)
	if err != nil || len(failed) != 2 {
		t.Fatalf("next read failures=%v err=%v", failed, err)
	}
	var count int64
	if err := db.Model(&model.FeedFailEvent{}).Where("student_id = ?", sid).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("unconsumed failures=%d err=%v", count, err)
	}
}

func TestAllowListReadRespectsZeroPartialAndSoftDeletedConfig(t *testing.T) {
	for _, tc := range []struct {
		name    string
		bits    uint16
		deleted bool
	}{
		{name: "zero"}, {name: "partial", bits: 48}, {name: "deleted", bits: model.DefaultPushConfig, deleted: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
			if err != nil {
				t.Fatal(err)
			}
			sqlDB, _ := db.DB()
			defer sqlDB.Close()
			if err := db.AutoMigrate(&model.FeedUserConfig{}); err != nil {
				t.Fatal(err)
			}
			config := model.FeedUserConfig{StudentId: "20260001", PushConfig: tc.bits}
			if err := db.Create(&config).Error; err != nil {
				t.Fatal(err)
			}
			if tc.deleted {
				if err := db.Delete(&config).Error; err != nil {
					t.Fatal(err)
				}
			}
			svc := NewFeedUserConfigService(nil, dao.NewFeedUserConfigDAO(db), nil)
			got, err := svc.FindOrCreateAllowList(context.Background(), config.StudentId)
			if err != nil {
				t.Fatal(err)
			}
			wantFeedbackAndLibrary := !tc.deleted && tc.bits == 48
			if got.Grade || got.Energy || got.Holiday || got.Muxi || got.FeedBack != wantFeedbackAndLibrary || got.Library == nil || *got.Library != wantFeedbackAndLibrary {
				t.Fatalf("config=%+v", got)
			}
		})
	}
}
