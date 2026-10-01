package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
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
	gormlogger "gorm.io/gorm/logger"
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
	mu       sync.Mutex
	cidCalls int
	pushes   []jpush.PushData
	targets  [][]string
	pushErr  error
	onPush   func()
}

func (c *deliveryPushClient) GetCID(context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cidCalls++
	return "stable-cid", nil
}
func (c *deliveryPushClient) Push(_ context.Context, tokens []string, data jpush.PushData) error {
	c.mu.Lock()
	c.targets = append(c.targets, append([]string(nil), tokens...))
	c.pushes = append(c.pushes, data)
	c.mu.Unlock()
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
			err = svc.dispatchBatch(ctx, []model.FeedPushDelivery{delivery})
			if scenario == "cancelled" {
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("dispatch error=%v, want context cancellation", err)
				}
			} else if err != nil {
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
				if err = svc.dispatchBatch(context.Background(), []model.FeedPushDelivery{delivery}); err != nil {
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

type pushTargetReadLogger struct {
	gormlogger.Interface
	configs atomic.Int64
	tokens  atomic.Int64
}

func (l *pushTargetReadLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	sql, _ := fc()
	if strings.HasPrefix(sql, "SELECT") {
		if strings.Contains(sql, "feed_user_configs") {
			l.configs.Add(1)
		}
		if strings.Contains(sql, "feed_user_tokens") {
			l.tokens.Add(1)
		}
	}
	l.Interface.Trace(ctx, begin, fc, err)
}

func TestPushDeliveryBatchTargetReads(t *testing.T) {
	for _, count := range []int{0, 1, 10, 21, 100} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			counter := &pushTargetReadLogger{Interface: gormlogger.Default.LogMode(gormlogger.Silent)}
			dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_busy_timeout=5000", t.Name())
			db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: counter})
			if err != nil {
				t.Fatal(err)
			}
			sqlDB, err := db.DB()
			if err != nil {
				t.Fatal(err)
			}
			// SQLite 单连接避免并发状态更新的表锁干扰查询次数验证。
			sqlDB.SetMaxOpenConns(1)
			t.Cleanup(func() { _ = sqlDB.Close() })
			if err = db.AutoMigrate(&model.FeedEvent{}, &model.FeedPushDelivery{}, &model.FeedUserConfig{}, &model.FeedUserToken{}); err != nil {
				t.Fatal(err)
			}
			for _, sid := range []string{"one", "two", "three"} {
				if err = db.Create(&model.FeedUserToken{StudentId: sid, Token: sid + "-token"}).Error; err != nil {
					t.Fatal(err)
				}
			}
			deliveries := make([]model.FeedPushDelivery, 0, count)
			for i := 0; i < count; i++ {
				sid := []string{"one", "two", "three"}[i%3]
				event := model.FeedEvent{StudentId: sid, Type: "muxi", Title: sid, DedupeKey: strconv.Itoa(i), ExtendFields: model.ExtendFields{}}
				if err = db.Create(&event).Error; err != nil {
					t.Fatal(err)
				}
				delivery := model.FeedPushDelivery{FeedEventID: event.ID, StudentId: sid, Status: model.PushDeliveryPending}
				if err = db.Create(&delivery).Error; err != nil {
					t.Fatal(err)
				}
				deliveries = append(deliveries, delivery)
			}
			counter.configs.Store(0)
			counter.tokens.Store(0)
			client := &deliveryPushClient{}
			wantSent := count
			if count == 21 {
				// 首组发送时关闭开关：本组沿用快照，后续组必须重新读取并抑制。
				wantSent = pushDeliveryMaxConcurrency
				var once sync.Once
				client.onPush = func() {
					once.Do(func() {
						configs := []model.FeedUserConfig{{StudentId: "one"}, {StudentId: "two"}, {StudentId: "three"}}
						if err := db.Create(&configs).Error; err != nil {
							t.Errorf("close push config: %v", err)
						}
					})
				}
			}
			configDAO := dao.NewFeedUserConfigDAO(db)
			push := NewPushService(client, configDAO, dao.NewUserFeedTokenDAO(db))
			svc := NewPushDeliveryService(dao.NewPushDeliveryDAO(db), configDAO, push, nil, zapx.NewZapLogger(zap.NewNop())).(*pushDeliveryService)
			if err = svc.dispatchBatch(context.Background(), deliveries); err != nil {
				t.Fatal(err)
			}
			groups := int64((count + pushDeliveryMaxConcurrency - 1) / pushDeliveryMaxConcurrency)
			if counter.configs.Load() != groups || counter.tokens.Load() != groups {
				t.Fatalf("config reads=%d, token reads=%d, want %d each", counter.configs.Load(), counter.tokens.Load(), groups)
			}
			var sent int64
			if err = db.Model(&model.FeedPushDelivery{}).Where("status = ?", model.PushDeliverySent).Count(&sent).Error; err != nil {
				t.Fatal(err)
			}
			if sent != int64(wantSent) || len(client.pushes) != wantSent || client.cidCalls != wantSent {
				t.Fatalf("sent=%d pushes=%d cidCalls=%d, want %d", sent, len(client.pushes), client.cidCalls, wantSent)
			}
			var suppressed int64
			if err = db.Model(&model.FeedPushDelivery{}).Where("status = ?", model.PushDeliverySuppressed).Count(&suppressed).Error; err != nil || suppressed != int64(count-wantSent) {
				t.Fatalf("suppressed=%d, want %d, err=%v", suppressed, count-wantSent, err)
			}
			for i, data := range client.pushes {
				if len(client.targets[i]) != 1 || client.targets[i][0] != data.Title+"-token" {
					t.Fatalf("wrong recipient: title=%s, tokens=%v", data.Title, client.targets[i])
				}
			}
		})
	}
}

func TestPushDeliveryPrefetchFailureDoesNotClaim(t *testing.T) {
	for _, scenario := range []string{"config_error", "token_error", "cancelled"} {
		t.Run(scenario, func(t *testing.T) {
			dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_busy_timeout=5000", t.Name())
			db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
			if err != nil {
				t.Fatal(err)
			}
			if err = db.AutoMigrate(&model.FeedPushDelivery{}, &model.FeedUserConfig{}, &model.FeedUserToken{}); err != nil {
				t.Fatal(err)
			}
			delivery := model.FeedPushDelivery{StudentId: "one", Status: model.PushDeliveryPending}
			if err = db.Create(&delivery).Error; err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch scenario {
			case "config_error":
				err = db.Migrator().DropTable(&model.FeedUserConfig{})
			case "token_error":
				err = db.Migrator().DropTable(&model.FeedUserToken{})
			case "cancelled":
				// 在配置预取完成时取消，不能进入认领阶段。
				err = db.Callback().Query().After("gorm:query").Register("test:cancel_prefetch", func(tx *gorm.DB) {
					if tx.Statement.Table == "feed_user_configs" {
						cancel()
					}
				})
			}
			if err != nil {
				t.Fatal(err)
			}
			client := &deliveryPushClient{}
			configDAO := dao.NewFeedUserConfigDAO(db)
			push := NewPushService(client, configDAO, dao.NewUserFeedTokenDAO(db))
			svc := NewPushDeliveryService(dao.NewPushDeliveryDAO(db), configDAO, push, nil, zapx.NewZapLogger(zap.NewNop())).(*pushDeliveryService)
			if err = svc.dispatchBatch(ctx, []model.FeedPushDelivery{delivery}); err == nil {
				t.Fatal("prefetch failure was ignored")
			}
			if err = db.First(&delivery, delivery.ID).Error; err != nil {
				t.Fatal(err)
			}
			if delivery.Status != model.PushDeliveryPending || delivery.Attempts != 0 || delivery.CID != "" || len(client.pushes) != 0 || client.cidCalls != 0 {
				t.Fatalf("prefetch failure changed delivery: %+v, pushes=%v", delivery, client.pushes)
			}
		})
	}
}

func TestPushTargetsSnapshotMatchesSinglePreparation(t *testing.T) {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_busy_timeout=5000", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&model.FeedUserConfig{}, &model.FeedUserToken{}); err != nil {
		t.Fatal(err)
	}
	configs := []model.FeedUserConfig{
		{StudentId: "enabled", PushConfig: model.DefaultPushConfig},
		{StudentId: "disabled", PushConfig: 0},
		{StudentId: "deleted", PushConfig: model.DefaultPushConfig},
	}
	if err = db.Create(&configs).Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Delete(&configs[2]).Error; err != nil {
		t.Fatal(err)
	}
	ids := []string{"enabled", "disabled", "deleted", "missing", "no_token", "enabled", "", " invalid "}
	for _, id := range ids[:4] {
		if err = db.Create(&model.FeedUserToken{StudentId: id, Token: id + "-token"}).Error; err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()
	push := NewPushService(&deliveryPushClient{}, dao.NewFeedUserConfigDAO(db), dao.NewUserFeedTokenDAO(db))
	targets, err := push.LoadPushTargets(ctx, ids)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets.users) != 5 {
		t.Fatalf("prefetched users=%d, want 5", len(targets.users))
	}
	for _, id := range ids {
		for _, label := range []string{"muxi", "library", "LIBRARY", "unknown"} {
			event := &domain.FeedEvent{StudentId: id, Type: label}
			single, singleReason, singleErr := push.PreparePushForDelivery(ctx, event)
			batch, batchReason, batchErr := targets.prepare(event)
			if singleErr != nil || batchErr != nil || singleReason != batchReason || fmt.Sprint(single) != fmt.Sprint(batch) {
				t.Fatalf("%s/%s: single=%v/%s/%v, batch=%v/%s/%v", id, label, single, singleReason, singleErr, batch, batchReason, batchErr)
			}
		}
	}
	// 预取后准备阶段不再访问数据库；缺失快照不能冒充缺失配置。
	if err = db.Migrator().DropTable(&model.FeedUserConfig{}, &model.FeedUserToken{}); err != nil {
		t.Fatal(err)
	}
	if prepared, _, err := targets.prepare(&domain.FeedEvent{StudentId: "enabled", Type: "muxi"}); err != nil || prepared == nil {
		t.Fatalf("in-memory preparation failed: %v, %v", prepared, err)
	}
	if _, _, err := targets.prepare(&domain.FeedEvent{StudentId: "not_prefetched", Type: "muxi"}); err == nil {
		t.Fatal("missing snapshot was treated as missing config")
	}
	if empty, err := push.LoadPushTargets(ctx, []string{"", " invalid "}); err != nil || len(empty.users) != 0 {
		t.Fatalf("invalid users triggered reads: %v, %v", empty, err)
	}
}
