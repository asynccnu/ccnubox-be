package dao

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/asynccnu/ccnubox-be/be-feed/repository/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGetPushFeedEventTreatsSoftDeletedEventAsNotFound(t *testing.T) {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_busy_timeout=5000", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err = db.AutoMigrate(&model.FeedEvent{}); err != nil {
		t.Fatalf("migrate feed event: %v", err)
	}
	event := model.FeedEvent{StudentId: "20260001", Type: "library", Title: "提醒"}
	if err = db.Create(&event).Error; err != nil {
		t.Fatalf("create feed event: %v", err)
	}
	if err = db.Delete(&event).Error; err != nil {
		t.Fatalf("soft delete feed event: %v", err)
	}

	repo := NewPushDeliveryDAO(db)
	_, err = repo.GetFeedEvent(context.Background(), event.ID)
	if !errors.Is(err, ErrFeedEventNotFound) {
		t.Fatalf("get soft-deleted feed event err=%v", err)
	}
}

func TestPushDeliveryPriority(t *testing.T) {
	for _, notificationType := range []string{"START_30", "END_10", "AWAY_60", "AWAY_80"} {
		event := model.FeedEvent{Type: "library", ExtendFields: model.ExtendFields{"notification_type": notificationType}}
		if priority := pushDeliveryPriority(event); priority <= 0 {
			t.Fatalf("notification type %s priority=%d", notificationType, priority)
		}
	}
	if priority := pushDeliveryPriority(model.FeedEvent{Type: "muxi"}); priority != 0 {
		t.Fatalf("normal message priority=%d", priority)
	}
}

func TestPushDeliveryListDuePrioritizesTimeSensitiveRows(t *testing.T) {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_busy_timeout=5000", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err = db.AutoMigrate(&model.FeedPushDelivery{}); err != nil {
		t.Fatalf("migrate push delivery: %v", err)
	}
	now := time.Now().Unix()
	rows := []model.FeedPushDelivery{
		{FeedEventID: 1, StudentId: "20260001", Status: model.PushDeliveryPending, Priority: 0, NextAttemptAt: now - 10},
		{FeedEventID: 2, StudentId: "20260002", Status: model.PushDeliveryPending, Priority: 100, NextAttemptAt: now},
		{FeedEventID: 3, StudentId: "20260003", Status: model.PushDeliveryPending, Priority: 100, NextAttemptAt: now - 1},
	}
	if err = db.Create(&rows).Error; err != nil {
		t.Fatalf("create push deliveries: %v", err)
	}

	due, err := NewPushDeliveryDAO(db).ListDue(context.Background(), now, len(rows))
	if err != nil {
		t.Fatalf("list due deliveries: %v", err)
	}
	want := []int64{rows[2].ID, rows[1].ID, rows[0].ID}
	if len(due) != len(want) {
		t.Fatalf("due count=%d, want=%d", len(due), len(want))
	}
	for i := range want {
		if due[i].ID != want[i] {
			t.Fatalf("due order=%v, want=%v", []int64{due[0].ID, due[1].ID, due[2].ID}, want)
		}
	}
}

func TestPushDeliveryRecoversSendingOnlyAtStartup(t *testing.T) {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_busy_timeout=5000", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err = db.AutoMigrate(&model.FeedPushDelivery{}); err != nil {
		t.Fatalf("migrate push delivery: %v", err)
	}
	repo := NewPushDeliveryDAO(db)
	ctx := context.Background()
	now := time.Now().Unix()
	rows := []model.FeedPushDelivery{
		{FeedEventID: 1, StudentId: "20260001", Status: model.PushDeliveryPending, NextAttemptAt: now},
		// 回拨 updated_at 到阈值之前以模拟孤儿记录；row3 模拟仍在推送中的活跃记录。
		{FeedEventID: 2, StudentId: "20260002", Status: model.PushDeliverySending, Attempts: 2, NextAttemptAt: now},
		{FeedEventID: 3, StudentId: "20260003", Status: model.PushDeliverySending, Attempts: 1, NextAttemptAt: now},
	}
	// 回拨 updated_at 到阈值之前以模拟孤儿记录；row3 模拟仍在推送中的活跃记录。
	if err = db.Create(&rows).Error; err != nil {
		t.Fatalf("create push deliveries: %v", err)
	}
	// GORM 在 Create/Update 时会强制重置 updated_at，故用 UpdateColumn 绕过钩子显式回拨。
	if err = db.Model(&model.FeedPushDelivery{}).Where("id = ?", rows[1].ID).
		UpdateColumn("updated_at", now-60).Error; err != nil {
		t.Fatalf("backdate updated_at: %v", err)
	}

	due, err := repo.ListDue(ctx, now, 10)
	if err != nil || len(due) != 1 || due[0].ID != rows[0].ID {
		t.Fatalf("due before recovery=%+v err=%v", due, err)
	}
	claimed, err := repo.Claim(ctx, rows[0].ID)
	if err != nil || !claimed {
		t.Fatalf("claim pending delivery: claimed=%v err=%v", claimed, err)
	}
	claimed, err = repo.Claim(ctx, rows[0].ID)
	if err != nil || claimed {
		t.Fatalf("claim sending delivery: claimed=%v err=%v", claimed, err)
	}

	// 只恢复超过超时阈值的 sending：陈旧记录被恢复，活跃记录不被误重置。
	if err = repo.RecoverSending(ctx, now); err != nil {
		t.Fatalf("recover sending deliveries: %v", err)
	}
	due, err = repo.ListDue(ctx, now, 10)
	if err != nil || len(due) != 1 || due[0].ID != rows[1].ID || due[0].Attempts != 2 {
		t.Fatalf("due after recovery=%+v err=%v", due, err)
	}
	var count int64
	if err = db.Model(&model.FeedPushDelivery{}).
		Where("id IN ? AND status = ?", []int64{rows[0].ID, rows[2].ID}, model.PushDeliverySending).
		Count(&count).Error; err != nil || count != 2 {
		t.Fatalf("active sending should be kept, count=%d err=%v", count, err)
	}
	if err = repo.MarkSent(ctx, rows[1].ID); err == nil {
		t.Fatal("pending delivery was finalized without being claimed")
	}
}

func TestTeamSuccessStoredWithPushDelivery(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared&_busy_timeout=5000", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.FeedEvent{}, &model.FeedUserConfig{}, &model.FeedPushDelivery{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.FeedUserConfig{StudentId: "20260001", PushConfig: model.DefaultPushConfig}).Error; err != nil {
		t.Fatal(err)
	}
	repo := NewFeedEventDAO(db)
	team := model.FeedEvent{StudentId: "20260001", Type: "library", DedupeKey: "team-1", ExtendFields: model.ExtendFields{"notification_type": "TEAM_SUCCESS", "team_id": "1"}}
	normal := model.FeedEvent{StudentId: "20260001", Type: "library", DedupeKey: "normal-1", ExtendFields: model.ExtendFields{"notification_type": "START_30"}}
	inserted, suppressed, err := repo.StoreFeedEvents(context.Background(), []model.FeedEvent{team, normal, team})
	if err != nil || suppressed != 0 || len(inserted) != 2 {
		t.Fatalf("inserted=%+v suppressed=%d err=%v", inserted, suppressed, err)
	}
	var count int64
	if err := db.Model(&model.FeedPushDelivery{}).Count(&count).Error; err != nil || count != 2 {
		t.Fatalf("deliveries=%d err=%v", count, err)
	}
	for _, event := range inserted {
		if err := db.Model(&model.FeedPushDelivery{}).Where("feed_event_id = ? AND status = ?", event.ID, model.PushDeliveryPending).Count(&count).Error; err != nil || count != 1 {
			t.Fatalf("event %d push delivery count=%d err=%v", event.ID, count, err)
		}
	}
	if err := db.Model(&model.FeedEvent{}).Count(&count).Error; err != nil || count != 2 {
		t.Fatalf("events=%d err=%v", count, err)
	}
}

func TestInvitationStoreIsAtomicAndDeduplicated(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	if err := db.AutoMigrate(&model.FeedEvent{}, &model.FeedPushDelivery{}, &model.FeedUserConfig{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.FeedUserConfig{StudentId: "A", PushConfig: 1 << model.LibraryPos}).Error; err != nil {
		t.Fatal(err)
	}
	event := model.FeedEvent{StudentId: "A", Type: "library", DedupeKey: "invitation", Source: "library", OccurredAt: 1, ExtendFields: model.ExtendFields{"notification_type": "TEAM_INVITATION", "team_id": "1", "expires_at": "2"}}
	repo := NewFeedEventDAO(db)
	// 过期只影响推送，不能阻止已被接受的事实落库。
	inserted, suppressed, err := repo.StoreFeedEvents(context.Background(), []model.FeedEvent{event, event})
	if err != nil || len(inserted) != 1 || suppressed != 0 {
		t.Fatalf("inserted=%v suppressed=%d err=%v", inserted, suppressed, err)
	}
	for _, m := range []any{&model.FeedEvent{}, &model.FeedPushDelivery{}} {
		var count int64
		if err := db.Model(m).Count(&count).Error; err != nil || count != 1 {
			t.Fatalf("model=%T count=%d err=%v", m, count, err)
		}
	}
	if priority := pushDeliveryPriority(event); priority != 0 {
		t.Fatalf("priority=%d", priority)
	}
	event.StudentId = "not-subscribed"
	inserted, suppressed, err = repo.StoreFeedEvents(context.Background(), []model.FeedEvent{event})
	if err != nil || len(inserted) != 0 || suppressed != 1 {
		t.Fatalf("inserted=%v suppressed=%d err=%v", inserted, suppressed, err)
	}
}

func TestPushTargetReadOnlyAndBounded(t *testing.T) {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_busy_timeout=5000", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&model.FeedUserConfig{}, &model.FeedUserToken{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	configs := []model.FeedUserConfig{{StudentId: "one", PushConfig: 0}, {StudentId: "deleted", PushConfig: model.DefaultPushConfig}}
	if err = db.Create(&configs).Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Delete(&configs[1]).Error; err != nil {
		t.Fatal(err)
	}
	configDAO := NewFeedUserConfigDAO(db)
	for _, tt := range []struct {
		id   string
		want PushConfigSnapshot
	}{
		{"one", PushConfigSnapshot{Exists: true}},
		{"deleted", PushConfigSnapshot{Exists: true, Deleted: true, Config: model.DefaultPushConfig}},
		{"missing", PushConfigSnapshot{}},
	} {
		got, err := configDAO.GetPushConfig(ctx, tt.id)
		if err != nil || got != tt.want {
			t.Fatalf("config[%s]=%+v, want=%+v, err=%v", tt.id, got, tt.want, err)
		}
	}
	var count int64
	if err = db.Model(&model.FeedUserConfig{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("read created config: count=%d, err=%v", count, err)
	}
	for i := 0; i < 6; i++ {
		token := model.FeedUserToken{StudentId: "one", Token: fmt.Sprintf("token-%d", i)}
		if err = db.Create(&token).Error; err != nil {
			t.Fatal(err)
		}
		if i == 5 {
			if err = db.Delete(&token).Error; err != nil {
				t.Fatal(err)
			}
		}
	}
	tokens, err := NewUserFeedTokenDAO(db).GetTokens(ctx, "one")
	if err != nil || len(tokens) != 4 || tokens[0] != "token-4" || tokens[3] != "token-1" {
		t.Fatalf("latest active tokens=%v, err=%v", tokens, err)
	}
}

func TestPushTargetsBatchReads(t *testing.T) {
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
	for _, id := range []string{"enabled", "disabled", "missing"} {
		for i := 0; i < 7; i++ {
			token := model.FeedUserToken{StudentId: id, Token: fmt.Sprintf("%s-%d", id, i)}
			if err = db.Create(&token).Error; err != nil {
				t.Fatal(err)
			}
			if i == 6 {
				if err = db.Delete(&token).Error; err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	// 固定同秒时间戳，验证按 ID 排序以及排名前排除软删除。
	if err = db.Unscoped().Model(&model.FeedUserToken{}).Where("1 = 1").Update("created_at", 100).Error; err != nil {
		t.Fatal(err)
	}
	reads := 0
	countRead := func(tx *gorm.DB) {
		if !tx.DryRun {
			reads++
		}
	}
	if err = db.Callback().Query().Before("gorm:query").Register("test:count_query", countRead); err != nil {
		t.Fatal(err)
	}
	if err = db.Callback().Row().Before("gorm:row").Register("test:count_row", countRead); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	configDAO := NewFeedUserConfigDAO(db)
	tokenDAO := NewUserFeedTokenDAO(db)
	ids := []string{"enabled", "disabled", "deleted", "missing", "enabled"}
	snapshots, err := configDAO.GetPushConfigs(ctx, ids)
	if err != nil {
		t.Fatal(err)
	}
	tokens, err := tokenDAO.GetTokensBatch(ctx, ids)
	if err != nil {
		t.Fatal(err)
	}
	if reads != 2 {
		t.Fatalf("batch reads=%d, want 2", reads)
	}
	for _, id := range ids {
		config, err := configDAO.GetPushConfig(ctx, id)
		if err != nil || snapshots[id] != config {
			t.Fatalf("config[%s]=%+v, want %+v, err=%v", id, snapshots[id], config, err)
		}
		single, err := tokenDAO.GetTokens(ctx, id)
		if err != nil || fmt.Sprint(tokens[id]) != fmt.Sprint(single) {
			t.Fatalf("tokens[%s]=%v, want %v, err=%v", id, tokens[id], single, err)
		}
		if id != "deleted" && (len(tokens[id]) != 4 || tokens[id][0] != id+"-5" || tokens[id][3] != id+"-2") {
			t.Fatalf("tokens[%s]=%v", id, tokens[id])
		}
	}
	reads = 0
	if result, err := configDAO.GetPushConfigs(ctx, nil); err != nil || len(result) != 0 {
		t.Fatalf("empty configs=%v, err=%v", result, err)
	}
	if result, err := tokenDAO.GetTokensBatch(ctx, []string{}); err != nil || len(result) != 0 {
		t.Fatalf("empty tokens=%v, err=%v", result, err)
	}
	if reads != 0 {
		t.Fatalf("empty batch made %d reads", reads)
	}
}
