package dao

import (
	"context"
	"errors"
	"fmt"
	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"os"
	"strings"
	"sync"
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
	if reads != 5 {
		t.Fatalf("batch reads=%d, want 1 config query + 4 unique-user token queries", reads)
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

// 设置 MYSQL57_TEST_DSN 后使用独立临时数据库，禁止复用或清空已有业务库。
func subscriptionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	var db *gorm.DB
	var err error
	if dsn := os.Getenv("MYSQL57_TEST_DSN"); dsn != "" {
		cfg, parseErr := mysqlDriver.ParseDSN(dsn)
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		cfg.DBName = ""
		admin, openErr := gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{})
		if openErr != nil {
			t.Fatal(openErr)
		}
		adminSQL, _ := admin.DB()
		t.Cleanup(func() { _ = adminSQL.Close() })
		var version string
		if err = admin.Raw("SELECT VERSION()").Scan(&version).Error; err != nil || !strings.HasPrefix(version, "5.7.") {
			t.Fatalf("requires MySQL 5.7, version=%s, err=%v", version, err)
		}
		name := fmt.Sprintf("ccnubox_feed_test_%d", time.Now().UnixNano())
		if err = admin.Exec("CREATE DATABASE " + name + " CHARACTER SET utf8mb4 COLLATE utf8mb4_bin").Error; err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = admin.Exec("DROP DATABASE " + name).Error })
		cfg.DBName = name
		db, err = gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{})
	} else {
		db, err = gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared&_busy_timeout=5000", t.Name())), &gorm.Config{})
	}
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err = db.AutoMigrate(&model.FeedEvent{}, &model.FeedFailEvent{}, &model.FeedPushDelivery{}, &model.FeedUserConfig{}, &model.FeedUserConfigChange{}, &model.FeedUserConfigRevisionAllocator{}, &model.FeedUserToken{}); err != nil {
		t.Fatal(err)
	}
	if err = db.Create(&model.FeedUserConfigRevisionAllocator{ID: model.FeedUserConfigRevisionAllocatorID}).Error; err != nil {
		t.Fatal(err)
	}
	return db
}

func TestFeedSubscriptionQueries(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config domain.PushConfigSnapshot
	}{
		{"zero", domain.PushConfigSnapshot{Exists: true}},
		{"feedback_library_only", domain.PushConfigSnapshot{Exists: true, Config: 48}},
		{"missing", domain.PushConfigSnapshot{}},
		{"deleted", domain.PushConfigSnapshot{Exists: true, Deleted: true, Config: model.DefaultPushConfig}},
		{"all", domain.PushConfigSnapshot{Exists: true, Config: model.DefaultPushConfig}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := subscriptionTestDB(t)
			ctx := context.Background()
			const sid = "20260001"
			if tc.config.Exists {
				config := model.FeedUserConfig{StudentId: sid, PushConfig: tc.config.Config}
				if err := db.Create(&config).Error; err != nil {
					t.Fatal(err)
				}
				if tc.config.Deleted {
					if err := db.Delete(&config).Error; err != nil {
						t.Fatal(err)
					}
					// 查询设置不能把软删除当作缺失，或重新创建成默认开启。
					if _, err := NewFeedUserConfigDAO(db).FindOrCreateUserFeedConfig(ctx, sid); err != nil {
						t.Fatal(err)
					}
				}
			}
			labels := append(domain.FeedTypes(), domain.FeedTypes()...)
			labels = append(labels, "unknown", "grade ", "gráde", "LIBRARY ")
			var want []int64
			for i, label := range labels {
				if i < 6 {
					label = strings.ToUpper(label)
				}
				event := model.FeedEvent{StudentId: sid, Type: label, DedupeKey: fmt.Sprint(i), ExtendFields: model.ExtendFields{}}
				if err := db.Create(&event).Error; err != nil {
					t.Fatal(err)
				}
				if tc.config.Allows(label) {
					want = append([]int64{event.ID}, want...)
				}
				failed := model.FeedFailEvent{StudentId: sid, Type: label, ExtendFields: model.ExtendFields{}}
				if err := db.Create(&failed).Error; err != nil {
					t.Fatal(err)
				}
			}
			// 固定创建时间，验证同秒消息以 ID 稳定倒序。
			if err := db.Model(&model.FeedEvent{}).Where("student_id = ?", sid).UpdateColumn("created_at", 100).Error; err != nil {
				t.Fatal(err)
			}
			deleted := model.FeedEvent{StudentId: sid, Type: "feedback", DedupeKey: "deleted"}
			other := model.FeedEvent{StudentId: "other", Type: "feedback", DedupeKey: "other"}
			if err := db.Create(&deleted).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Delete(&deleted).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Create(&other).Error; err != nil {
				t.Fatal(err)
			}
			got, err := NewFeedEventDAO(db).GetFeedEventsByStudentId(ctx, sid)
			if err != nil || len(got) != len(want) {
				t.Fatalf("events=%v want=%v err=%v", got, want, err)
			}
			for i := range want {
				if got[i].ID != want[i] {
					t.Fatalf("order[%d]=%d want=%d", i, got[i].ID, want[i])
				}
			}
			failDAO := NewFeedFailEventDAO(db)
			failed, err := failDAO.GetFeedFailEventsByStudentId(ctx, sid)
			if err != nil || len(failed) != len(want) {
				t.Fatalf("failed=%v wantCount=%d err=%v", failed, len(want), err)
			}
			ids := make([]int64, len(failed))
			for i := range failed {
				ids[i] = failed[i].ID
			}
			if err := failDAO.DelFeedFailEventsByStudentId(ctx, sid, ids); err != nil {
				t.Fatal(err)
			}
			var remaining int64
			if err := db.Model(&model.FeedFailEvent{}).Count(&remaining).Error; err != nil || remaining != int64(len(labels)-len(want)) {
				t.Fatalf("hidden failed messages=%d err=%v", remaining, err)
			}
		})
	}
}

func TestFeedFailConsumptionRestrictsIDsStudentAndSubscription(t *testing.T) {
	db := subscriptionTestDB(t)
	ctx := context.Background()
	const sid = "20260001"
	if err := db.Create(&model.FeedUserConfig{StudentId: sid, PushConfig: 1 << model.GradePos}).Error; err != nil {
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
	failDAO := NewFeedFailEventDAO(db)
	// 空 ID 列表必须直接退出，不能退化成按学号删除所有记录。
	if err := failDAO.DelFeedFailEventsByStudentId(ctx, sid, nil); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&model.FeedFailEvent{}).Count(&count).Error; err != nil || count != int64(len(failures)) {
		t.Fatalf("empty IDs consumed failures: count=%d err=%v", count, err)
	}
	ids := make([]int64, len(failures))
	for i := range failures {
		ids[i] = failures[i].ID
	}
	// 即使传入其他账号及当前关闭类型的 ID，也只能消费该账号当前允许的记录。
	if err := failDAO.DelFeedFailEventsByStudentId(ctx, sid, ids); err != nil {
		t.Fatal(err)
	}
	var remaining []model.FeedFailEvent
	if err := db.Order("id ASC").Find(&remaining).Error; err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 2 || remaining[0].ID != failures[1].ID || remaining[1].ID != failures[2].ID {
		t.Fatalf("unexpected remaining failures: %v", remaining)
	}
}

func TestFeedSubscriptionFiltersBeforeLimit(t *testing.T) {
	db := subscriptionTestDB(t)
	const sid = "20260001"
	if err := db.Create(&model.FeedUserConfig{StudentId: sid, PushConfig: 1 << model.GradePos}).Error; err != nil {
		t.Fatal(err)
	}
	var allowed []int64
	for i := 0; i < 45; i++ {
		label := "MUXI"
		if i < 25 {
			label = "GrAdE"
		}
		event := model.FeedEvent{StudentId: sid, Type: label, DedupeKey: fmt.Sprint(i)}
		if err := db.Create(&event).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Model(&event).UpdateColumn("created_at", i+1).Error; err != nil {
			t.Fatal(err)
		}
		if i < 25 {
			allowed = append([]int64{event.ID}, allowed...)
		}
	}
	events, err := NewFeedEventDAO(db).GetFeedEventsByStudentId(context.Background(), sid)
	if err != nil || len(events) != 20 {
		t.Fatalf("count=%d err=%v", len(events), err)
	}
	for i := range events {
		if events[i].ID != allowed[i] {
			t.Fatalf("wrong allowed order at %d", i)
		}
	}
}

func TestChangeConfigCancelsOnlyClosedPendingAndNeverRestores(t *testing.T) {
	db := subscriptionTestDB(t)
	ctx := context.Background()
	const sid = "20260001"
	config := model.FeedUserConfig{StudentId: sid, PushConfig: model.DefaultPushConfig, LibraryRevision: 7}
	if err := db.Create(&config).Error; err != nil {
		t.Fatal(err)
	}
	var deliveries []model.FeedPushDelivery
	var want []string
	for _, recipient := range []string{sid, "other"} {
		for _, label := range domain.FeedTypes() {
			for _, state := range []string{model.PushDeliveryPending, model.PushDeliverySending, model.PushDeliverySent, model.PushDeliveryFailed, model.PushDeliverySuppressed} {
				event := model.FeedEvent{StudentId: recipient, Type: strings.ToUpper(label), DedupeKey: label + state}
				if err := db.Create(&event).Error; err != nil {
					t.Fatal(err)
				}
				delivery := model.FeedPushDelivery{StudentId: recipient, FeedEventID: event.ID, Status: state}
				if err := db.Create(&delivery).Error; err != nil {
					t.Fatal(err)
				}
				deliveries = append(deliveries, delivery)
				if recipient == sid && (label == "grade" || label == "feedback") && state == model.PushDeliveryPending {
					state = model.PushDeliverySuppressed
				}
				want = append(want, state)
			}
		}
	}
	configDAO := NewFeedUserConfigDAO(db)
	changed, err := configDAO.ChangeConfigBits(ctx, sid, map[int]bool{model.GradePos: false, model.FeedBackPos: false}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if changed.PushConfig != model.DefaultPushConfig&^((1<<model.GradePos)|(1<<model.FeedBackPos)) || changed.LibraryRevision != 7 {
		t.Fatalf("unexpected config: %+v", changed)
	}
	// 再开启只恢复历史可见性，不能恢复 suppressed 投递。
	if _, err := configDAO.ChangeConfigBits(ctx, sid, map[int]bool{model.GradePos: true, model.FeedBackPos: true}, nil); err != nil {
		t.Fatal(err)
	}
	for i, delivery := range deliveries {
		var got model.FeedPushDelivery
		if err := db.First(&got, delivery.ID).Error; err != nil {
			t.Fatal(err)
		}
		if got.Status != want[i] {
			t.Fatalf("delivery=%d status=%s want=%s", got.ID, got.Status, want[i])
		}
		if delivery.Status == model.PushDeliveryPending && got.Status == model.PushDeliverySuppressed {
			if got.LastError != "suppressed_by_allow_list" {
				t.Fatalf("reason=%s", got.LastError)
			}
			if claimed, err := NewPushDeliveryDAO(db).Claim(ctx, got.ID); err != nil || claimed {
				t.Fatalf("cancelled task claimed=%v err=%v", claimed, err)
			}
		}
	}
	var count int64
	if err := db.Model(&model.FeedEvent{}).Count(&count).Error; err != nil || count != int64(len(deliveries)) {
		t.Fatalf("history lost: count=%d err=%v", count, err)
	}
}

func TestChangeConfigRollbackAndMissingDefaults(t *testing.T) {
	for _, missing := range []bool{false, true} {
		for _, fail := range []bool{false, true} {
			t.Run(fmt.Sprintf("missing_%v_fail_%v", missing, fail), func(t *testing.T) {
				db := subscriptionTestDB(t)
				ctx := context.Background()
				const sid = "20260001"
				config := model.FeedUserConfig{StudentId: sid, PushConfig: model.DefaultPushConfig}
				if !missing {
					if err := db.Create(&config).Error; err != nil {
						t.Fatal(err)
					}
				}
				events, _, err := NewFeedEventDAO(db).StoreFeedEvents(ctx, []model.FeedEvent{{StudentId: sid, Type: "GRADE", DedupeKey: "one"}})
				if err != nil || len(events) != 1 {
					t.Fatalf("store: %v %v", events, err)
				}
				if fail {
					if err := db.Callback().Update().Before("gorm:update").Register("test:cancel_failure", func(tx *gorm.DB) {
						if tx.Statement.Table == "feed_push_deliveries" {
							tx.AddError(errors.New("cancel failed"))
						}
					}); err != nil {
						t.Fatal(err)
					}
				}
				library := false
				_, err = NewFeedUserConfigDAO(db).ChangeConfigBits(ctx, sid, map[int]bool{model.GradePos: false}, &library)
				if (err != nil) != fail {
					t.Fatalf("change err=%v wantFailure=%v", err, fail)
				}
				snapshot, err := NewFeedUserConfigDAO(db).GetPushConfig(ctx, sid)
				if err != nil {
					t.Fatal(err)
				}
				var delivery model.FeedPushDelivery
				if err := db.Where("feed_event_id = ?", events[0].ID).First(&delivery).Error; err != nil {
					t.Fatal(err)
				}
				wantStatus := model.PushDeliverySuppressed
				if fail {
					wantStatus = model.PushDeliveryPending
					if snapshot.Exists == missing || (!missing && snapshot.Config != model.DefaultPushConfig) {
						t.Fatalf("config not rolled back: %+v", snapshot)
					}
					var changes int64
					if err := db.Model(&model.FeedUserConfigChange{}).Count(&changes).Error; err != nil || changes != 0 {
						t.Fatalf("changes=%d err=%v", changes, err)
					}
				} else if snapshot.Allows("grade") || snapshot.Allows("library") {
					t.Fatalf("still enabled: %+v", snapshot)
				}
				if delivery.Status != wantStatus {
					t.Fatalf("status=%s want=%s", delivery.Status, wantStatus)
				}
			})
		}
	}
}

func TestTokensBatchFailureDiscardsPartialResults(t *testing.T) {
	db := subscriptionTestDB(t)
	for _, sid := range []string{"first", "second"} {
		for i := 0; i < 6; i++ {
			token := model.FeedUserToken{StudentId: sid, Token: fmt.Sprint(i)}
			if err := db.Create(&token).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Model(&token).UpdateColumn("created_at", 100).Error; err != nil {
				t.Fatal(err)
			}
			if i == 5 {
				if err := db.Delete(&token).Error; err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	repo := NewUserFeedTokenDAO(db)
	result, err := repo.GetTokensBatch(context.Background(), []string{"first", "first", "second"})
	if err != nil {
		t.Fatal(err)
	}
	for _, sid := range []string{"first", "second"} {
		if fmt.Sprint(result[sid]) != "[4 3 2 1]" {
			t.Fatalf("tokens[%s]=%v", sid, result[sid])
		}
	}
	queries := 0
	if err := db.Callback().Query().Before("gorm:query").Register("test:partial_token_failure", func(tx *gorm.DB) {
		if tx.Statement.Table == "feed_user_tokens" {
			queries++
			if queries == 2 {
				tx.AddError(errors.New("second user query failed"))
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	result, err = repo.GetTokensBatch(context.Background(), []string{"first", "first", "second"})
	if err == nil || result != nil || queries != 2 {
		t.Fatalf("partial result=%v err=%v queries=%d", result, err, queries)
	}
}

func TestSubscriptionMySQL57LocksAndPlans(t *testing.T) {
	if os.Getenv("MYSQL57_TEST_DSN") == "" {
		t.Skip("需要隔离的 MySQL 5.7 测试环境")
	}
	t.Run("claim_waits_for_cancellation_commit", func(t *testing.T) {
		db := subscriptionTestDB(t)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		const sid = "20260001"
		if err := db.Create(&model.FeedUserConfig{StudentId: sid, PushConfig: model.DefaultPushConfig}).Error; err != nil {
			t.Fatal(err)
		}
		events, _, err := NewFeedEventDAO(db).StoreFeedEvents(ctx, []model.FeedEvent{{StudentId: sid, Type: "grade", DedupeKey: "one"}})
		if err != nil {
			t.Fatal(err)
		}
		var delivery model.FeedPushDelivery
		if err := db.Where("feed_event_id = ?", events[0].ID).First(&delivery).Error; err != nil {
			t.Fatal(err)
		}
		updated, release := make(chan struct{}), make(chan struct{})
		var releaseOnce sync.Once
		unblock := func() { releaseOnce.Do(func() { close(release) }) }
		defer unblock()
		if err := db.Callback().Update().After("gorm:update").Register("test:pause_cancel_commit", func(tx *gorm.DB) {
			values, ok := tx.Statement.Dest.(map[string]any)
			if ok && tx.Statement.Table == "feed_push_deliveries" && values["status"] == model.PushDeliverySuppressed {
				close(updated)
				<-release
			}
		}); err != nil {
			t.Fatal(err)
		}
		changed := make(chan error, 1)
		go func() {
			_, err := NewFeedUserConfigDAO(db).ChangeConfigBits(ctx, sid, map[int]bool{model.GradePos: false}, nil)
			changed <- err
		}()
		select {
		case <-updated:
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
		type claimResult struct {
			claimed bool
			err     error
		}
		claimed := make(chan claimResult, 1)
		go func() { ok, err := NewPushDeliveryDAO(db).Claim(ctx, delivery.ID); claimed <- claimResult{ok, err} }()
		select {
		case result := <-claimed:
			t.Fatalf("claim did not wait for commit: %+v", result)
		case <-time.After(150 * time.Millisecond):
		}
		unblock()
		if err := <-changed; err != nil {
			t.Fatal(err)
		}
		result := <-claimed
		if result.err != nil || result.claimed {
			t.Fatalf("cancelled task claimed=%v err=%v", result.claimed, result.err)
		}
	})
	t.Run("execution_plans", func(t *testing.T) {
		db := subscriptionTestDB(t)
		var events []model.FeedEvent
		var tokens []model.FeedUserToken
		for i := 0; i < 1000; i++ {
			sid := fmt.Sprintf("student-%03d", i/10)
			events = append(events, model.FeedEvent{StudentId: sid, Type: "grade", DedupeKey: fmt.Sprint(i)})
			tokens = append(tokens, model.FeedUserToken{StudentId: sid, Token: fmt.Sprint(i)})
		}
		if err := db.CreateInBatches(&events, 100).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.CreateInBatches(&tokens, 100).Error; err != nil {
			t.Fatal(err)
		}
		var deliveries []model.FeedPushDelivery
		for _, event := range events {
			deliveries = append(deliveries, model.FeedPushDelivery{StudentId: event.StudentId, FeedEventID: event.ID, Status: model.PushDeliveryPending})
		}
		if err := db.CreateInBatches(&deliveries, 100).Error; err != nil {
			t.Fatal(err)
		}
		type planRow struct {
			Table string
			Type  string
			Key   string
			Rows  int64
			Extra string
		}
		checkPlan := func(name, sql, table, index string) {
			t.Helper()
			var plan []planRow
			if err := db.Raw("EXPLAIN " + sql).Scan(&plan).Error; err != nil {
				t.Fatal(err)
			}
			found := false
			for _, row := range plan {
				t.Logf("%s: table=%s access=%s key=%s rows=%d extra=%s", name, row.Table, row.Type, row.Key, row.Rows, row.Extra)
				if row.Table == table {
					found = true
					if index != "" && row.Key != index {
						t.Errorf("%s index=%s want=%s", name, row.Key, index)
					}
				}
			}
			if !found {
				t.Fatalf("%s table not found in plan", name)
			}
		}
		inboxSQL := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return withFeedSubscription(tx.Model(&model.FeedEvent{}), "feed_events").Select("feed_events.*").Where("feed_events.student_id = ?", "student-042").Order("feed_events.created_at DESC, feed_events.id DESC").Limit(20).Find(&[]model.FeedEvent{})
		})
		checkPlan("inbox", inboxSQL, "feed_events", "idx_feed_events_inbox")
		cancelSQL := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
			events := tx.Session(&gorm.Session{NewDB: true}).Unscoped().Model(&model.FeedEvent{}).Select("id").Where("student_id = ? AND (("+feedTypeMaskSQL("type")+") & ?) <> 0", "student-042", 1<<model.GradePos)
			return tx.Model(&model.FeedPushDelivery{}).Where("student_id = ? AND status = ? AND feed_event_id IN (?)", "student-042", model.PushDeliveryPending, events).Updates(map[string]any{"status": model.PushDeliverySuppressed})
		})
		checkPlan("cancel", cancelSQL, "feed_push_deliveries", "idx_feed_push_deliveries_student_id")
		tokenSQL := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return tx.Model(&model.FeedUserToken{}).Select("token").Where("student_id = ?", "student-042").Order("created_at DESC, id DESC").Limit(4).Find(&[]string{})
		})
		checkPlan("token_before_manual_index", tokenSQL, "feed_user_tokens", "")
		// 仅在独立测试库验证候选索引；生产环境必须单独评估，不能放进启动迁移。
		if err := db.Exec("CREATE INDEX idx_feed_user_tokens_recipient_recent ON feed_user_tokens (student_id(255), deleted_at, created_at, id)").Error; err != nil {
			t.Fatal(err)
		}
		checkPlan("token_after_manual_index", tokenSQL, "feed_user_tokens", "idx_feed_user_tokens_recipient_recent")
		started := time.Now()
		if _, err := NewFeedUserConfigDAO(db).ChangeConfigBits(context.Background(), "student-042", map[int]bool{model.GradePos: false}, nil); err != nil {
			t.Fatal(err)
		}
		t.Logf("1000-row fixture, cancel 10 pending tasks: %s", time.Since(started))
	})
}

func TestClosedSubscriptionsKeepOrdinaryStorageIdempotent(t *testing.T) {
	db := subscriptionTestDB(t)
	const sid = "20260001"
	if err := db.Create(&model.FeedUserConfig{StudentId: sid, PushConfig: 0}).Error; err != nil {
		t.Fatal(err)
	}
	var events []model.FeedEvent
	for _, label := range domain.FeedTypes() {
		event := model.FeedEvent{StudentId: sid, Type: strings.ToUpper(label), DedupeKey: label}
		events = append(events, event, event)
	}
	inserted, suppressed, err := NewFeedEventDAO(db).StoreFeedEvents(context.Background(), events)
	if err != nil || len(inserted) != 5 || suppressed != 2 {
		t.Fatalf("inserted=%d suppressed=%d err=%v", len(inserted), suppressed, err)
	}
	for _, event := range inserted {
		if event.Type != domain.NormalizeFeedType(event.Type) {
			t.Fatalf("noncanonical type: %s", event.Type)
		}
	}
	var count int64
	for _, m := range []any{&model.FeedEvent{}, &model.FeedPushDelivery{}} {
		if err := db.Model(m).Count(&count).Error; err != nil || count != 5 {
			t.Fatalf("model=%T count=%d err=%v", m, count, err)
		}
	}
	visible, err := NewFeedEventDAO(db).GetFeedEventsByStudentId(context.Background(), sid)
	if err != nil || len(visible) != 0 {
		t.Fatalf("disabled events visible: %v err=%v", visible, err)
	}
	if err := db.Callback().Create().Before("gorm:create").Register("test:delivery_insert_failure", func(tx *gorm.DB) {
		if tx.Statement.Table == "feed_push_deliveries" {
			tx.AddError(errors.New("delivery insert failed"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	_, _, err = NewFeedEventDAO(db).StoreFeedEvents(context.Background(), []model.FeedEvent{{StudentId: sid, Type: "grade", DedupeKey: "must-rollback"}})
	if err == nil {
		t.Fatal("delivery insert failure ignored")
	}
	if err := db.Model(&model.FeedEvent{}).Count(&count).Error; err != nil || count != 5 {
		t.Fatalf("orphan event count=%d err=%v", count, err)
	}
}

func TestChangeConfigRevivesSoftDeletedConfig(t *testing.T) {
	libraryOff, libraryOn := false, true
	for _, tc := range []struct {
		name       string
		library    *bool
		wantConfig uint16
	}{
		{name: "omitted_library"},
		{name: "disabled_library", library: &libraryOff},
		{name: "enabled_library", library: &libraryOn, wantConfig: 1 << model.LibraryPos},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := subscriptionTestDB(t)
			ctx := context.Background()
			const sid = "20260001"
			const oldRevision int64 = 7
			config := model.FeedUserConfig{StudentId: sid, PushConfig: model.DefaultPushConfig, LibraryRevision: oldRevision}
			if err := db.Create(&config).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Model(&model.FeedUserConfigRevisionAllocator{}).Where("id = ?", model.FeedUserConfigRevisionAllocatorID).UpdateColumn("revision", oldRevision).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Create(&model.FeedUserConfigChange{Revision: oldRevision, StudentId: sid, LibraryEnabled: true}).Error; err != nil {
				t.Fatal(err)
			}
			event := model.FeedEvent{StudentId: sid, Type: "grade", DedupeKey: "before-delete"}
			if err := db.Create(&event).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Delete(&config).Error; err != nil {
				t.Fatal(err)
			}
			// 软删除期间按全关处理，列表不返回任何消息。
			if visible, err := NewFeedEventDAO(db).GetFeedEventsByStudentId(ctx, sid); err != nil || len(visible) != 0 {
				t.Fatalf("deleted config visible=%v err=%v", visible, err)
			}
			configDAO := NewFeedUserConfigDAO(db)
			before, err := configDAO.GetPushConfig(ctx, sid)
			if err != nil || before.EffectiveConfig() != 0 {
				t.Fatalf("deleted config=%+v err=%v", before, err)
			}
			// 恢复时以有效全关状态为起点，未传入的 library 不能复活原始位图中的开启位。
			bits := map[int]bool{model.GradePos: false, model.MuxiPos: false, model.HolidayPos: false, model.EnergyPos: false, model.FeedBackPos: false}
			changed, err := configDAO.ChangeConfigBits(ctx, sid, bits, tc.library)
			if err != nil {
				t.Fatal(err)
			}
			if changed.ID != config.ID || changed.DeletedAt.Valid || changed.PushConfig != tc.wantConfig {
				t.Fatalf("config not revived as requested: %+v", changed)
			}
			snapshot, err := configDAO.GetPushConfig(ctx, sid)
			if err != nil || !snapshot.Exists || snapshot.Deleted || snapshot.Config != tc.wantConfig {
				t.Fatalf("snapshot=%+v err=%v", snapshot, err)
			}
			wantRevision := oldRevision
			wantChanges := 0
			if tc.wantConfig != 0 {
				wantRevision++
				wantChanges = 1
			}
			if changed.LibraryRevision != wantRevision {
				t.Fatalf("library revision=%d want=%d", changed.LibraryRevision, wantRevision)
			}
			changes, err := configDAO.ListLibraryPreferenceChanges(ctx, oldRevision, 10)
			if err != nil || len(changes) != wantChanges {
				t.Fatalf("library changes=%v err=%v", changes, err)
			}
			if wantChanges == 1 && (changes[0].Revision != wantRevision || changes[0].StudentId != sid || !changes[0].LibraryEnabled) {
				t.Fatalf("library enable change=%+v", changes[0])
			}
			if visible, err := NewFeedEventDAO(db).GetFeedEventsByStudentId(ctx, sid); err != nil || len(visible) != 0 {
				t.Fatalf("closed type visible=%v err=%v", visible, err)
			}
			changed, err = configDAO.ChangeConfigBits(ctx, sid, map[int]bool{model.GradePos: true}, nil)
			if err != nil || changed.PushConfig != tc.wantConfig|(1<<model.GradePos) || changed.LibraryRevision != wantRevision {
				t.Fatalf("grade change affected library: %+v err=%v", changed, err)
			}
			if visible, err := NewFeedEventDAO(db).GetFeedEventsByStudentId(ctx, sid); err != nil || len(visible) != 1 {
				t.Fatalf("history not restored: %v err=%v", visible, err)
			}
		})
	}
}
