package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	"github.com/asynccnu/ccnubox-be/be-feed/pkg/jpush"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/model"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger/zapx"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestLibraryPushExpired(t *testing.T) {
	// 固定时钟，精确覆盖截止前一秒、截止时刻和截止后一秒。
	now := time.Unix(1700000000, 0)
	future := strconv.FormatInt(now.Unix()+1, 10)
	past := strconv.FormatInt(now.Unix()-1, 10)

	reminders := []struct {
		name             string
		eventType        string
		notificationType string
		field            string
	}{
		{"START_30", "library", "START_30", "start_at"},
		{"END_10", "library", "END_10", "end_at"},
		{"AWAY_60", "library", "AWAY_60", "end_at"},
		{"AWAY_80", "library", "AWAY_80", "end_at"},
		{"normalized_START_30", "LiBrArY", " \tstart_30\n", "start_at"},
		{"normalized_END_10", "LiBrArY", " \tend_10\n", "end_at"},
		{"normalized_AWAY_60", "LiBrArY", " \taway_60\n", "end_at"},
		{"normalized_AWAY_80", "LiBrArY", " \taway_80\n", "end_at"},
	}
	deadlines := []struct {
		name    string
		value   string
		missing bool
		want    bool
	}{
		{name: "future", value: future},
		{name: "at_deadline", value: strconv.FormatInt(now.Unix(), 10), want: true},
		{name: "past", value: past, want: true},
		{name: "missing", missing: true, want: true},
		{name: "empty", value: "", want: true},
		{name: "invalid", value: "invalid", want: true},
		{name: "overflow", value: "9223372036854775808", want: true},
		{name: "zero", value: "0", want: true},
		{name: "negative", value: "-1", want: true},
	}
	for _, reminder := range reminders {
		t.Run(reminder.name, func(t *testing.T) {
			for _, deadline := range deadlines {
				t.Run(deadline.name, func(t *testing.T) {
					// 另一时间字段与预期相反，防止误选字段或回退到另一字段。
					other := future
					if !deadline.want {
						other = past
					}
					fields := model.ExtendFields{
						"notification_type": reminder.notificationType,
						"start_at":          other,
						"end_at":            other,
					}
					if deadline.missing {
						delete(fields, reminder.field)
					} else {
						fields[reminder.field] = deadline.value
					}
					event := model.FeedEvent{Type: reminder.eventType, ExtendFields: fields}
					if got := libraryPushExpired(&event, now); got != deadline.want {
						t.Fatalf("libraryPushExpired(%+v) = %v, want %v", event, got, deadline.want)
					}
				})
			}
		})
	}

	// 普通通知、事实类消息及未知通知类型不受时间字段限制。
	unaffected := []struct {
		name             string
		eventType        string
		notificationType string
	}{
		{"ordinary", "muxi", ""},
		{"ordinary_START_30", "muxi", "START_30"},
		{"ordinary_END_10", "muxi", "END_10"},
		{"ordinary_AWAY_60", "muxi", "AWAY_60"},
		{"ordinary_AWAY_80", "muxi", "AWAY_80"},
		{"RESERVATION_DISCOVERED", "library", "RESERVATION_DISCOVERED"},
		{"BREACH", "library", "BREACH"},
		{"BLACKLISTED", "library", "BLACKLISTED"},
		{"unknown", "library", "UNKNOWN"},
		{"missing_notification_type", "library", ""},
	}
	for _, tt := range unaffected {
		t.Run(tt.name, func(t *testing.T) {
			for _, deadline := range deadlines {
				t.Run(deadline.name, func(t *testing.T) {
					fields := model.ExtendFields{}
					if tt.notificationType != "" {
						fields["notification_type"] = tt.notificationType
					}
					if !deadline.missing {
						fields["start_at"] = deadline.value
						fields["end_at"] = deadline.value
					}
					event := model.FeedEvent{Type: tt.eventType, ExtendFields: fields}
					if libraryPushExpired(&event, now) {
						t.Fatalf("libraryPushExpired(%+v) = true, want false", event)
					}
				})
			}
		})
	}
}

type deliveryPushClient struct {
	cidCalls int
	pushes   []jpush.PushData
	pushErr  error
	onPush   func()
}

func (c *deliveryPushClient) GetCID(context.Context) (string, error) {
	c.cidCalls++
	return "stable-cid", nil
}
func (c *deliveryPushClient) Push(_ context.Context, _ []string, data jpush.PushData) error {
	c.pushes = append(c.pushes, data)
	if c.onPush != nil {
		c.onPush()
	}
	return c.pushErr
}

type failingLibraryGate struct{ dao.FeedUserConfigDAO }

func (g *failingLibraryGate) IsLibraryEnabled(context.Context, string) (bool, error) {
	return false, errors.New("library gate unavailable")
}

type deliveryFaultDAO struct {
	dao.PushDeliveryDAO
	claimFalse bool
	saveErr    bool
	sentErr    bool
}

func (d *deliveryFaultDAO) Claim(ctx context.Context, id int64) (bool, error) {
	if d.claimFalse {
		return false, nil
	}
	return d.PushDeliveryDAO.Claim(ctx, id)
}
func (d *deliveryFaultDAO) SaveCID(ctx context.Context, id int64, cid string) error {
	if d.saveErr {
		return errors.New("save cid failed")
	}
	return d.PushDeliveryDAO.SaveCID(ctx, id, cid)
}
func (d *deliveryFaultDAO) MarkSent(ctx context.Context, id int64) error {
	if d.sentErr {
		return errors.New("mark sent failed")
	}
	return d.PushDeliveryDAO.MarkSent(ctx, id)
}

func TestPushAllowedSnapshot(t *testing.T) {
	for _, tt := range []struct {
		name, label string
		config      dao.PushConfigSnapshot
		want        bool
	}{
		{"ordinary_missing", "muxi", dao.PushConfigSnapshot{}, true},
		{"library_missing", "library", dao.PushConfigSnapshot{}, false},
		{"explicit_zero", "muxi", dao.PushConfigSnapshot{Exists: true}, false},
		{"deleted", "muxi", dao.PushConfigSnapshot{Exists: true, Deleted: true, Config: model.DefaultPushConfig}, false},
		{"enabled", "library", dao.PushConfigSnapshot{Exists: true, Config: model.DefaultPushConfig}, true},
		{"unknown", "OTHER", dao.PushConfigSnapshot{}, false},
		{"case_sensitive", "LIBRARY", dao.PushConfigSnapshot{Exists: true, Config: model.DefaultPushConfig}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := pushAllowed(tt.label, tt.config); got != tt.want {
				t.Fatalf("pushAllowed=%v, want %v", got, tt.want)
			}
		})
	}
}

func TestPushDeliveryPersistenceBranches(t *testing.T) {
	for _, scenario := range []string{"sent", "claim_false", "missing_event", "no_token", "disabled", "expired", "save_cid_error", "push_error", "mark_sent_error", "cancelled", "recipient_mismatch", "library_closed_before_send", "library_gate_error"} {
		t.Run(scenario, func(t *testing.T) {
			dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_busy_timeout=5000", t.Name())
			db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
			if err != nil {
				t.Fatal(err)
			}
			if err = db.AutoMigrate(&model.FeedEvent{}, &model.FeedPushDelivery{}, &model.FeedUserConfig{}, &model.FeedUserToken{}); err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			sid := "20260001"
			event := model.FeedEvent{StudentId: sid, Type: "muxi", Title: "通知", ExtendFields: model.ExtendFields{}}
			if scenario == "expired" || scenario == "library_closed_before_send" || scenario == "library_gate_error" {
				event.Type = "library"
				event.ExtendFields = model.ExtendFields{"notification_type": "START_30", "start_at": strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10)}
				if scenario == "expired" {
					event.ExtendFields["start_at"] = "1"
				}
			}
			if scenario == "recipient_mismatch" {
				event.StudentId = "different"
			}
			if scenario != "missing_event" {
				if err = db.Create(&event).Error; err != nil {
					t.Fatal(err)
				}
			}
			delivery := model.FeedPushDelivery{FeedEventID: event.ID, StudentId: sid, Status: model.PushDeliveryPending}
			if scenario == "missing_event" {
				delivery.FeedEventID = 999
			}
			if err = db.Create(&delivery).Error; err != nil {
				t.Fatal(err)
			}
			if scenario != "no_token" {
				if err = db.Create(&model.FeedUserToken{StudentId: sid, Token: "target"}).Error; err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "disabled" || scenario == "library_closed_before_send" || scenario == "library_gate_error" {
				value := uint16(0)
				if scenario == "library_closed_before_send" || scenario == "library_gate_error" {
					value = model.DefaultPushConfig
				}
				if err = db.Create(&model.FeedUserConfig{StudentId: sid, PushConfig: value}).Error; err != nil {
					t.Fatal(err)
				}
			}
			client := &deliveryPushClient{}
			if scenario == "push_error" {
				client.pushErr = errors.New("push failed")
			}
			if scenario == "library_closed_before_send" {
				client.onPush = func() { t.Error("library closed but JPush called") }
			}
			fault := &deliveryFaultDAO{PushDeliveryDAO: dao.NewPushDeliveryDAO(db), claimFalse: scenario == "claim_false", saveErr: scenario == "save_cid_error", sentErr: scenario == "mark_sent_error"}
			push := NewPushService(client, dao.NewFeedUserConfigDAO(db), dao.NewUserFeedTokenDAO(db))
			svc := NewPushDeliveryService(fault, dao.NewFeedUserConfigDAO(db), push, nil, zapx.NewZapLogger(zap.NewNop())).(*pushDeliveryService)
			if scenario == "library_gate_error" {
				svc.gate = &failingLibraryGate{FeedUserConfigDAO: dao.NewFeedUserConfigDAO(db)}
			}
			if scenario == "library_closed_before_send" {
				// 在获取 CID 之后关闭偏好，验证最终门禁。
				client.onPush = nil
				clientClose := &closingPushClient{deliveryPushClient: client, db: db, sid: sid}
				svc.push = NewPushService(clientClose, dao.NewFeedUserConfigDAO(db), dao.NewUserFeedTokenDAO(db))
			}
			if scenario == "cancelled" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				client.onPush = cancel
			}
			if err = svc.dispatchOne(ctx, delivery); err != nil {
				t.Fatal(err)
			}
			if err = db.First(&delivery, delivery.ID).Error; err != nil {
				t.Fatal(err)
			}
			wantStatus := model.PushDeliverySuppressed
			switch scenario {
			case "sent", "cancelled":
				wantStatus = model.PushDeliverySent
			case "claim_false":
				wantStatus = model.PushDeliveryPending
			case "save_cid_error", "push_error", "mark_sent_error", "recipient_mismatch", "library_gate_error":
				wantStatus = model.PushDeliveryPending
				if delivery.Attempts != 1 {
					t.Fatalf("attempts=%d", delivery.Attempts)
				}
			}
			if delivery.Status != wantStatus {
				t.Fatalf("status=%s, want %s", delivery.Status, wantStatus)
			}
			shouldPush := scenario == "sent" || scenario == "push_error" || scenario == "mark_sent_error" || scenario == "cancelled"
			if (len(client.pushes) != 0) != shouldPush {
				t.Fatalf("pushes=%v", client.pushes)
			}
			if scenario == "save_cid_error" && delivery.CID != "" {
				t.Fatalf("CID saved despite error: %q", delivery.CID)
			}
			if scenario == "mark_sent_error" {
				fault.sentErr = false
				if err = svc.dispatchOne(context.Background(), delivery); err != nil {
					t.Fatal(err)
				}
				if client.cidCalls != 1 || len(client.pushes) != 2 || client.pushes[1].Cid != "stable-cid" {
					t.Fatalf("CID not reused: calls=%d pushes=%v", client.cidCalls, client.pushes)
				}
			}
		})
	}
}

type closingPushClient struct {
	*deliveryPushClient
	db  *gorm.DB
	sid string
}

func (c *closingPushClient) GetCID(ctx context.Context) (string, error) {
	cid, err := c.deliveryPushClient.GetCID(ctx)
	if err == nil {
		err = c.db.Model(&model.FeedUserConfig{}).Where("student_id = ?", c.sid).Update("push_config", 0).Error
	}
	return cid, err
}

func TestPreparePushReadOnly(t *testing.T) {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_busy_timeout=5000", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&model.FeedUserConfig{}, &model.FeedUserToken{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	ids := []string{"one", "two"}
	for _, id := range ids {
		if err = db.Create(&model.FeedUserToken{StudentId: id, Token: id + "-token"}).Error; err != nil {
			t.Fatal(err)
		}
	}
	push := NewPushService(&deliveryPushClient{}, dao.NewFeedUserConfigDAO(db), dao.NewUserFeedTokenDAO(db))
	ordinary := &domain.FeedEvent{StudentId: "one", Type: "muxi"}
	library := &domain.FeedEvent{StudentId: "one", Type: "library"}
	prepared, reason, err := push.PreparePushForDelivery(ctx, ordinary)
	if err != nil || prepared == nil || reason != "" {
		t.Fatalf("ordinary missing config: prepared=%v reason=%s err=%v", prepared, reason, err)
	}
	prepared, reason, err = push.PreparePushForDelivery(ctx, library)
	if err != nil || prepared != nil || reason != "suppressed_by_allow_list" {
		t.Fatalf("library missing config: prepared=%v reason=%s err=%v", prepared, reason, err)
	}
	var count int64
	if err = db.Model(&model.FeedUserConfig{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("config was created: count=%d err=%v", count, err)
	}
	if err = db.Migrator().DropTable(&model.FeedUserConfig{}); err != nil {
		t.Fatal(err)
	}
	if _, _, err = push.PreparePushForDelivery(ctx, ordinary); err == nil {
		t.Fatal("config query error was treated as default allow")
	}
}
