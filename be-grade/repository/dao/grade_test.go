package dao

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/asynccnu/ccnubox-be/be-grade/domain"
	"github.com/asynccnu/ccnubox-be/be-grade/repository/model"
	"github.com/asynccnu/ccnubox-be/common/pkg/testx"
	"github.com/go-sql-driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newGradeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	if db := testx.MySQL(t); db != nil {
		return db
	}
	dsn := filepath.Join(t.TempDir(), "grades.db") + "?_journal_mode=WAL&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

func newGradeTestDAO(t *testing.T) (*gorm.DB, GradeDAO) {
	t.Helper()
	db := newGradeTestDB(t)
	if err := InitTables(db); err != nil {
		t.Fatal(err)
	}
	return db, NewGradeDAO(db)
}

func snapshot(student string, grades ...model.Grade) GradeSnapshot {
	return GradeSnapshot{StudentID: student, SourceSystem: "undergraduate", Grades: grades, Complete: true, ObservedAt: time.Now()}
}

func syncGrades(t *testing.T, d GradeDAO, student string, grades ...model.Grade) GradeSyncResult {
	t.Helper()
	result, err := d.SyncGrades(context.Background(), snapshot(student, grades...))
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func countRows(t *testing.T, db *gorm.DB, table any, want int64) {
	t.Helper()
	var count int64
	if err := db.Model(table).Count(&count).Error; err != nil || count != want {
		t.Fatalf("table=%T count=%d want=%d err=%v", table, count, want, err)
	}
}

func TestSyncGradesBaselineEmpty(t *testing.T) {
	db, d := newGradeTestDAO(t)
	if got := syncGrades(t, d, "student"); len(got.Notify) != 0 {
		t.Fatal(got)
	}
	countRows(t, db, &model.GradeSyncState{}, 1)
	got := syncGrades(t, d, "student", model.Grade{StudentId: "student", JxbId: "class", Cj: 0})
	if len(got.Notify) != 1 || got.Notify[0].ScoreVersion != 1 || got.Notify[0].ChangeVersion != 1 {
		t.Fatal(got)
	}
	countRows(t, db, &model.GradeNotificationOutbox{}, 1)
	var row model.GradeNotificationOutbox
	if err := db.First(&row).Error; err != nil {
		t.Fatal(err)
	}
	event, err := domain.ParseGradeNotification(row)
	if err != nil || event.Title != "成绩发布提醒" || event.ExtendFields["change_reason"] != "first_published" {
		t.Fatal(event, err)
	}
}

func TestSyncGradesVersionsAndMetadata(t *testing.T) {
	db, d := newGradeTestDAO(t)
	grade := model.Grade{StudentId: "student", JxbId: "class", Kcmc: "课程", Cj: 80, Xnm: 2020, Xqm: 1,
		RegularGradePercent: "未知", FinalGradePercent: "未知"}
	baseline := syncGrades(t, d, "student", grade)
	if len(baseline.Changed) != 1 || len(baseline.Notify) != 0 {
		t.Fatal(baseline)
	}
	id := baseline.Changed[0].NotificationIdentity()
	if len(id) != 32 {
		t.Fatal(id)
	}
	countRows(t, db, &model.GradeNotificationOutbox{}, 0)
	grade.Kcmc, grade.KcId, grade.Xf, grade.Jd = "改名", "详情ID更新", 3, 4
	grade.Kcxzmc, grade.Kclbmc, grade.Kcbj = "必修", "专业", "主修"
	metadata := syncGrades(t, d, "student", grade)
	if len(metadata.Changed) != 1 || len(metadata.Notify) != 0 || metadata.Changed[0].ScoreVersion != 1 || metadata.Changed[0].ChangeVersion != 2 {
		t.Fatal(metadata)
	}
	for i, score := range []float32{90, 80, 0} {
		grade.Cj = score
		result := syncGrades(t, d, "student", grade, grade)
		if len(result.Notify) != 1 || result.Notify[0].NotificationIdentity() != id || result.Notify[0].ScoreVersion != int64(i+2) || result.Notify[0].ChangeVersion != int64(i+3) {
			t.Fatal(result)
		}
		if same := syncGrades(t, d, "student", grade); len(same.Notify) != 0 || len(same.Changed) != 0 {
			t.Fatal(same)
		}
	}
	var rows []model.GradeNotificationOutbox
	if err := db.Order("id").Find(&rows).Error; err != nil || len(rows) != 3 {
		t.Fatalf("rows=%v err=%v", rows, err)
	}
	keys := map[string]bool{}
	for _, row := range rows {
		event, err := domain.ParseGradeNotification(row)
		if err != nil || keys[row.DedupeKey] || event.Title != "成绩更新提醒" || !strings.Contains(event.Url, "2020-1") {
			t.Fatalf("event=%v err=%v", event, err)
		}
		keys[row.DedupeKey] = true
	}
	// 快照中缺少课程不能删除本地历史成绩。
	syncGrades(t, d, "student")
	countRows(t, db, &model.Grade{}, 1)
}

func TestUpdateGradeDetailsRejectsStaleAndPreservesScoreVersion(t *testing.T) {
	db, d := newGradeTestDAO(t)
	base := syncGrades(t, d, "student", model.Grade{StudentId: "student", JxbId: "class", Cj: 80}).Changed[0]
	detail := base
	detail.Cj, detail.RegularGrade, detail.FinalGrade = 20, 80, 80
	detail.RegularGradePercent, detail.FinalGradePercent = "30", "70"
	detail.ChangeVersion = 0
	if err := d.UpdateGradeDetails(context.Background(), []model.Grade{detail}); !errors.Is(err, ErrGradeVersionChanged) {
		t.Fatal(err)
	}
	detail.ChangeVersion = base.ChangeVersion
	if err := d.UpdateGradeDetails(context.Background(), []model.Grade{detail}); err != nil {
		t.Fatal(err)
	}
	if err := d.UpdateGradeDetails(context.Background(), []model.Grade{detail}); err != nil {
		t.Fatal(err)
	}
	got, err := d.FindGrades(context.Background(), "student", 0, 0)
	if err != nil || len(got) != 1 || got[0].Cj != 80 || got[0].ScoreVersion != 1 || got[0].ChangeVersion != 2 || got[0].RegularGrade != 80 {
		t.Fatalf("got=%v err=%v", got, err)
	}
	metadata := got[0]
	metadata.Kcmc, metadata.RegularGrade = "改名", 0
	syncGrades(t, d, "student", metadata)
	got, _ = d.FindGrades(context.Background(), "student", 0, 0)
	if got[0].RegularGrade != 80 || got[0].ChangeVersion != 3 {
		t.Fatal(got)
	}
	countRows(t, db, &model.GradeNotificationOutbox{}, 0)
	if err := db.Delete(&got[0]).Error; err != nil {
		t.Fatal(err)
	}
	if err := d.UpdateGradeDetails(context.Background(), []model.Grade{detail}); err != nil {
		t.Fatal(err)
	}
	countRows(t, db, &model.Grade{}, 0)
}

func TestSyncGradesRollsBackAndRebuildsResultsOnRetry(t *testing.T) {
	for _, retry := range []bool{false, true} {
		db, d := newGradeTestDAO(t)
		syncGrades(t, d, "student")
		calls := 0
		if err := db.Callback().Create().Before("gorm:create").Register("test:outbox-failure", func(tx *gorm.DB) {
			if tx.Statement.Table == "grade_notification_outbox" {
				calls++
				if retry && calls == 1 {
					tx.AddError(&mysql.MySQLError{Number: 1213, Message: "deadlock"})
				} else if !retry {
					tx.AddError(errors.New("outbox unavailable"))
				}
			}
		}); err != nil {
			t.Fatal(err)
		}
		result, err := d.SyncGrades(context.Background(), snapshot("student", model.Grade{StudentId: "student", JxbId: "class", Cj: 0}))
		if retry {
			if err != nil || calls != 2 || len(result.Notify) != 1 || len(result.Changed) != 1 || result.Notify[0].ScoreVersion != 1 {
				t.Fatalf("result=%v err=%v calls=%d", result, err, calls)
			}
			countRows(t, db, &model.GradeNotificationOutbox{}, 1)
		} else {
			if err == nil || len(result.Notify) != 0 || len(result.Changed) != 0 {
				t.Fatalf("result=%v err=%v", result, err)
			}
			countRows(t, db, &model.Grade{}, 0)
			countRows(t, db, &model.GradeIdentityAlias{}, 0)
			countRows(t, db, &model.GradeNotificationOutbox{}, 0)
		}
	}
}

func TestSyncGradesSerializesConcurrentVersions(t *testing.T) {
	db, d := newGradeTestDAO(t)
	syncGrades(t, d, "student", model.Grade{StudentId: "student", JxbId: "class", Cj: 70})
	results := make(chan GradeSyncResult, 2)
	errs := make(chan error, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for _, point := range []float32{2, 3} {
		wg.Add(1)
		go func(jd float32) {
			defer wg.Done()
			<-start
			result, err := d.SyncGrades(context.Background(), snapshot("student", model.Grade{StudentId: "student", JxbId: "class", Cj: 80, Jd: jd}))
			results <- result
			errs <- err
		}(point)
	}
	close(start)
	wg.Wait()
	versions := []int64{}
	for range 2 {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
		versions = append(versions, (<-results).Changed[0].ChangeVersion)
	}
	slices.Sort(versions)
	if !slices.Equal(versions, []int64{2, 3}) {
		t.Fatal(versions)
	}
	countRows(t, db, &model.GradeNotificationOutbox{}, 1)
}

func TestSyncGradesIdentityAliasesAndQuarantine(t *testing.T) {
	db, d := newGradeTestDAO(t)
	base := syncGrades(t, d, "student", model.Grade{StudentId: "student", JxbId: "old", Kcmc: "课程", Cj: 80}).Changed[0]
	// 模拟人工确认后的新标识映射，不能按 KcId 或名称自动合并。
	alias := model.GradeIdentityAlias{StudentID: "student", SourceSystem: "undergraduate", AliasKind: "jxb_id", AliasValue: "real", NotificationID: base.NotificationIdentity()}
	if err := db.Create(&alias).Error; err != nil {
		t.Fatal(err)
	}
	base.JxbId, base.Kcmc = "real", "课程改名"
	got := syncGrades(t, d, "student", base)
	if len(got.Changed) != 1 || len(got.Notify) != 0 || got.Changed[0].NotificationIdentity() != base.NotificationIdentity() || got.Changed[0].ScoreVersion != 1 {
		t.Fatal(got)
	}
	countRows(t, db, &model.Grade{}, 1)
	bad := snapshot("student", model.Grade{StudentId: "student", JxbId: "", Kcmc: "歧义", Cj: 90})
	if result, err := d.SyncGrades(context.Background(), bad); err != nil || len(result.Skipped) != 1 {
		t.Fatal(result, err)
	}
	countRows(t, db, &model.GradeSyncIssue{}, 1)
	countRows(t, db, &model.GradeNotificationOutbox{}, 0)
}

func TestSyncGradesRejectsConflictingBatchAndStudent(t *testing.T) {
	db, d := newGradeTestDAO(t)
	grade := model.Grade{StudentId: "student", JxbId: "class", Cj: 80}
	conflict := grade
	conflict.Cj = 90
	if _, err := d.SyncGrades(context.Background(), snapshot("student", grade, conflict)); err == nil {
		t.Fatal("conflict accepted")
	}
	if _, err := d.SyncGrades(context.Background(), snapshot("other", grade)); err == nil {
		t.Fatal("cross-student snapshot accepted")
	}
	countRows(t, db, &model.GradeSyncState{}, 0)
}

func TestGradeSchemaRejectsWrongTypesAndDefaults(t *testing.T) {
	for _, fields := range []string{
		"notification_id varchar(16) NULL, score_version bigint NOT NULL DEFAULT 0",
		"notification_id varchar(32) NULL DEFAULT '', score_version bigint NOT NULL DEFAULT 0",
		"notification_id varchar(32) NULL DEFAULT 'NULL', score_version bigint NOT NULL DEFAULT 0",
		"notification_id varchar(32) NULL, score_version bigint NULL DEFAULT 0",
		"notification_id varchar(32) NULL, score_version bigint NOT NULL DEFAULT 1",
	} {
		db := newGradeTestDB(t)
		if err := db.Exec("CREATE TABLE grades (student_id varchar(100), jxb_id varchar(100), " + fields + ")").Error; err != nil {
			t.Fatal(err)
		}
		if err := InitTables(db); err == nil {
			t.Fatal("invalid schema accepted", fields)
		}
		if err := MigrateGradeNotifications(context.Background(), db); err == nil {
			t.Fatal("invalid schema silently repaired", fields)
		}
		if db.Migrator().HasTable(&model.GradeNotificationOutbox{}) {
			t.Fatal("invalid schema created auxiliary tables")
		}
	}
}

func TestConcurrentNewUserInitializationRemainsSilent(t *testing.T) {
	db, d := newGradeTestDAO(t)
	start := make(chan struct{})
	errs := make(chan error, 2)
	results := make(chan GradeSyncResult, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			result, err := d.SyncGrades(context.Background(), snapshot("new-user", model.Grade{StudentId: "new-user", JxbId: "class", Cj: 80}))
			results <- result
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	for range 2 {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
		if got := <-results; len(got.Notify) != 0 {
			t.Fatal(got)
		}
	}
	countRows(t, db, &model.Grade{}, 1)
	countRows(t, db, &model.GradeSyncState{}, 1)
	countRows(t, db, &model.GradeIdentityAlias{}, 1)
	countRows(t, db, &model.GradeNotificationOutbox{}, 0)
}

func TestNotificationMigrationIsSilentAndIdempotent(t *testing.T) {
	db, d := newGradeTestDAO(t)
	id, _ := model.NewNotificationID()
	rows := []model.Grade{
		{StudentId: "legacy", JxbId: "class", Cj: 80, ChangeVersion: 19},
		{StudentId: "legacy-pseudo", JxbId: "课程20201", Kcmc: "课程", Xnm: 2020, Xqm: 1, Cj: 0, ChangeVersion: 5},
		{StudentId: "active", JxbId: "active-class", Cj: 90, NotificationID: &id, ScoreVersion: 3, ChangeVersion: 7},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	state := model.GradeSyncState{StudentID: "active", InitializedAt: 10, LastSuccessAt: 20}
	alias := model.GradeIdentityAlias{StudentID: "active", SourceSystem: "undergraduate", AliasKind: "jxb_id", AliasValue: "active-class", NotificationID: id}
	event, err := domain.NewGradeNotification(rows[2], "score_changed", time.Now().Unix())
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []any{&state, &alias, &event} {
		if err := db.Create(value).Error; err != nil {
			t.Fatal(err)
		}
	}
	// 迁移和启动不得通过 ORM 读取或回填业务数据；元数据查询走 Row/Raw，不受影响。
	rejectDataAccess := func(tx *gorm.DB) {
		switch tx.Statement.Table {
		case "grades", "grade_identity_aliases", "grade_sync_states", "grade_notification_outbox":
			tx.AddError(errors.New("migration must not access business rows"))
		}
	}
	if err := db.Callback().Query().Before("gorm:query").Register("test:schema-only", rejectDataAccess); err != nil {
		t.Fatal(err)
	}
	if err := db.Callback().Create().Before("gorm:create").Register("test:schema-only", rejectDataAccess); err != nil {
		t.Fatal(err)
	}
	if err := db.Callback().Update().Before("gorm:update").Register("test:schema-only", rejectDataAccess); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := MigrateGradeNotifications(context.Background(), db); err != nil {
			t.Fatal(err)
		}
	}
	if err := InitTables(db); err != nil {
		t.Fatal(err)
	}
	db.Callback().Query().Remove("test:schema-only")
	db.Callback().Create().Remove("test:schema-only")
	db.Callback().Update().Remove("test:schema-only")
	for _, want := range rows {
		got, err := d.FindGrades(context.Background(), want.StudentId, 0, 0)
		if err != nil || len(got) != 1 || !reflect.DeepEqual(got[0], want) {
			t.Fatalf("got=%+v want=%+v err=%v", got, want, err)
		}
	}
	countRows(t, db, &model.GradeSyncState{}, 1)
	countRows(t, db, &model.GradeIdentityAlias{}, 1)
	countRows(t, db, &model.GradeNotificationOutbox{}, 1)
	var storedState model.GradeSyncState
	if err := db.First(&storedState).Error; err != nil || storedState != state {
		t.Fatal(storedState, err)
	}
	var storedEvent model.GradeNotificationOutbox
	if err := db.First(&storedEvent).Error; err != nil || storedEvent.Payload != event.Payload || storedEvent.DedupeKey != event.DedupeKey {
		t.Fatal(storedEvent, err)
	}
}

func TestNotificationMigrationFromLegacySchema(t *testing.T) {
	db := newGradeTestDB(t)
	if err := db.Exec("CREATE TABLE grades (student_id varchar(100) NOT NULL, jxb_id varchar(100) NOT NULL, cj real, change_version bigint, PRIMARY KEY(student_id, jxb_id))").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO grades VALUES ('2023212345', 'class', 0, 8), ('unknown-student', '课程20201', NULL, 9)").Error; err != nil {
		t.Fatal(err)
	}
	if err := InitTables(db); err == nil {
		t.Fatal("legacy schema accepted without DDL")
	}
	for range 2 {
		if err := MigrateGradeNotifications(context.Background(), db); err != nil {
			t.Fatal(err)
		}
	}
	var grades []model.Grade
	if err := db.Order("student_id").Find(&grades).Error; err != nil || len(grades) != 2 {
		t.Fatal(grades, err)
	}
	for i, grade := range grades {
		if grade.NotificationID != nil || grade.ScoreVersion != 0 || grade.ChangeVersion != int64(i+8) {
			t.Fatal(grade)
		}
	}
	var nullScores int64
	if err := db.Model(&model.Grade{}).Where("cj IS NULL").Count(&nullScores).Error; err != nil || nullScores != 1 {
		t.Fatal(nullScores, err)
	}
	// 只添加通知字段，不顺便通过 AutoMigrate 改造已有业务列。
	if db.Migrator().HasColumn(&model.Grade{}, "Kcmc") {
		t.Fatal("migration implicitly altered unrelated business columns")
	}
	columns, err := gradeNotificationColumns(db)
	if err != nil {
		t.Fatal(err)
	}
	if nullable, known := columns["notification_id"].Nullable(); !known || !nullable {
		t.Fatal("legacy identities must remain nullable")
	}
	if nullable, known := columns["score_version"].Nullable(); !known || nullable {
		t.Fatal("score version must remain non-null")
	}
	countRows(t, db, &model.GradeIdentityAlias{}, 0)
	countRows(t, db, &model.GradeSyncState{}, 0)
	countRows(t, db, &model.GradeNotificationOutbox{}, 0)
}

func TestNotificationMigrationRejectsPartialSchemaWithoutRepair(t *testing.T) {
	db := newGradeTestDB(t)
	if err := db.Exec("CREATE TABLE grades (student_id varchar(100), jxb_id varchar(100), notification_id varchar(32) NULL)").Error; err != nil {
		t.Fatal(err)
	}
	if err := MigrateGradeNotifications(context.Background(), db); err == nil {
		t.Fatal("partial migration silently repaired")
	}
	if db.Migrator().HasColumn(&model.Grade{}, "ScoreVersion") {
		t.Fatal("partial schema was changed")
	}
}

func TestInitTablesDoesNotRepairMissingNotificationIndex(t *testing.T) {
	db, _ := newGradeTestDAO(t)
	if err := db.Migrator().DropIndex(&model.Grade{}, gradeNotificationIdentityIndex); err != nil {
		t.Fatal(err)
	}
	if err := InitTables(db); err == nil {
		t.Fatal("missing unique index accepted")
	}
	if db.Migrator().HasIndex(&model.Grade{}, gradeNotificationIdentityIndex) {
		t.Fatal("startup silently created an index on existing grades")
	}
}

func TestConfirmedPseudoIdentityKeepsVersionsAndExistingEvents(t *testing.T) {
	db, d := newGradeTestDAO(t)
	id, _ := model.NewNotificationID()
	old := model.Grade{StudentId: "student", JxbId: "课程20201", Kcmc: "课程", Xnm: 2020, Xqm: 1, Cj: 80, NotificationID: &id, ChangeVersion: 7, ScoreVersion: 2}
	if err := db.Create(&old).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.GradeSyncState{StudentID: "student", InitializedAt: 1, LastSuccessAt: 1}).Error; err != nil {
		t.Fatal(err)
	}
	for _, alias := range []string{old.JxbId, "real"} {
		if err := db.Create(&model.GradeIdentityAlias{StudentID: "student", SourceSystem: "undergraduate", AliasKind: "jxb_id", AliasValue: alias, NotificationID: id}).Error; err != nil {
			t.Fatal(err)
		}
	}
	event, _ := domain.NewGradeNotification(old, "score_changed", time.Now().Unix())
	if err := db.Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	old.JxbId = "real"
	got := syncGrades(t, d, "student", old)
	if len(got.Notify) != 0 || len(got.Changed) != 1 || got.Changed[0].NotificationIdentity() != id || got.Changed[0].ScoreVersion != 2 || got.Changed[0].ChangeVersion != 8 {
		t.Fatal(got)
	}
	countRows(t, db, &model.Grade{}, 1)
	countRows(t, db, &model.GradeNotificationOutbox{}, 1)
	old.Cj = 90
	got = syncGrades(t, d, "student", old)
	if len(got.Notify) != 1 || got.Notify[0].ScoreVersion != 3 || got.Notify[0].NotificationIdentity() != id {
		t.Fatal(got)
	}
	countRows(t, db, &model.GradeNotificationOutbox{}, 2)
}

func TestGradeNotificationMySQLConstraints(t *testing.T) {
	db, d := newGradeTestDAO(t)
	grade := syncGrades(t, d, "student", model.Grade{StudentId: "student", JxbId: "class", Cj: 80}).Changed[0]
	duplicate := grade
	duplicate.JxbId = "different-class"
	if err := db.Create(&duplicate).Error; err == nil {
		t.Fatal("duplicate notification identity accepted")
	}
	for _, class := range []string{"legacy-one", "legacy-two"} {
		if err := db.Create(&model.Grade{StudentId: "student", JxbId: class}).Error; err != nil {
			t.Fatal("multiple NULL notification identities rejected", err)
		}
	}
	id, _ := model.NewNotificationID()
	if err := db.Exec("INSERT INTO grades (student_id, jxb_id, notification_id, score_version) VALUES (?, ?, ?, NULL)", "student", "null-version", id).Error; err == nil {
		t.Fatal("null business version accepted")
	}
	grade.Cj = 90
	syncGrades(t, d, "student", grade)
	var event model.GradeNotificationOutbox
	if err := db.First(&event).Error; err != nil {
		t.Fatal(err)
	}
	event.ID = 0
	if err := db.Create(&event).Error; err == nil {
		t.Fatal("duplicate outbox dedupe key accepted")
	}
	countRows(t, db, &model.Grade{}, 3)
	countRows(t, db, &model.GradeNotificationOutbox{}, 1)
}

func TestLegacyGradesStayUntrackedUntilScoreChanges(t *testing.T) {
	db, d := newGradeTestDAO(t)
	legacy := model.Grade{StudentId: "legacy", JxbId: "class", Kcmc: "旧课程", Cj: 80, ChangeVersion: 19,
		RegularGradePercent: "30", RegularGrade: 70, FinalGradePercent: "70", FinalGrade: 85}
	second := model.Grade{StudentId: "legacy", JxbId: "other", Cj: 0, ChangeVersion: 5}
	if err := db.Create(&[]model.Grade{legacy, second}).Error; err != nil {
		t.Fatal(err)
	}
	updates := 0
	if err := db.Callback().Update().Before("gorm:update").Register("test:count-grade-writes", func(tx *gorm.DB) {
		if tx.Statement.Table == "grades" {
			updates++
		}
	}); err != nil {
		t.Fatal(err)
	}
	if got := syncGrades(t, d, "legacy", legacy, second); len(got.Changed) != 0 || len(got.Notify) != 0 || updates != 0 {
		t.Fatal(got, updates)
	}
	legacy.Kcmc, legacy.KcId, legacy.Xf = "课程改名", "new-detail-id", 3
	metadata := syncGrades(t, d, "legacy", legacy, second)
	if len(metadata.Changed) != 1 || len(metadata.Notify) != 0 || metadata.Changed[0].NotificationID != nil || metadata.Changed[0].ScoreVersion != 0 || metadata.Changed[0].ChangeVersion != 20 {
		t.Fatal(metadata)
	}
	detail := metadata.Changed[0]
	detail.RegularGrade, detail.FinalGrade = 88, 90
	if err := d.UpdateGradeDetails(context.Background(), []model.Grade{detail}); err != nil {
		t.Fatal(err)
	}
	got, err := d.FindGrades(context.Background(), "legacy", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, grade := range got {
		if grade.NotificationID != nil || grade.ScoreVersion != 0 {
			t.Fatal(grade)
		}
		if grade.JxbId == "class" && (grade.ChangeVersion != 21 || grade.RegularGrade != 88 || grade.FinalGrade != 90) {
			t.Fatal(grade)
		}
	}
	if updates != 2 {
		t.Fatal("unexpected grade writes", updates)
	}
	countRows(t, db, &model.GradeIdentityAlias{}, 0)
	countRows(t, db, &model.GradeNotificationOutbox{}, 0)
}

func TestLegacyFirstRefreshInheritsBaselineAndActivatesOnlyChanges(t *testing.T) {
	db, d := newGradeTestDAO(t)
	legacy := model.Grade{StudentId: "legacy", JxbId: "class", Cj: 80, ChangeVersion: 19}
	unchanged := model.Grade{StudentId: "legacy", JxbId: "unchanged", Cj: 60, ChangeVersion: 7}
	if err := db.Create(&[]model.Grade{legacy, unchanged}).Error; err != nil {
		t.Fatal(err)
	}
	legacy.Cj = 90
	oldTerm := model.Grade{StudentId: "legacy", JxbId: "old-term-new-score", Xnm: 2020, Xqm: 1, Cj: 0}
	got := syncGrades(t, d, "legacy", unchanged, legacy, oldTerm)
	if len(got.Notify) != 2 || len(got.Changed) != 2 || got.Notify[0].ScoreVersion != 1 || got.Notify[0].ChangeVersion != 20 || got.Notify[1].ScoreVersion != 1 {
		t.Fatal(got)
	}
	id := got.Notify[0].NotificationIdentity()
	if len(id) != 32 || got.Notify[1].NotificationIdentity() == id {
		t.Fatal(got)
	}
	if repeated := syncGrades(t, d, "legacy", unchanged, legacy, oldTerm); len(repeated.Notify) != 0 || len(repeated.Changed) != 0 {
		t.Fatal(repeated)
	}
	var events []model.GradeNotificationOutbox
	if err := db.Order("id").Find(&events).Error; err != nil || len(events) != 2 {
		t.Fatal(events, err)
	}
	for i, reason := range []string{"score_changed", "first_published"} {
		event, err := domain.ParseGradeNotification(events[i])
		if err != nil || event.ExtendFields["change_reason"] != reason {
			t.Fatal(event, err)
		}
	}
	legacy.Cj = 80
	next := syncGrades(t, d, "legacy", unchanged, legacy, oldTerm)
	if len(next.Notify) != 1 || next.Notify[0].NotificationIdentity() != id || next.Notify[0].ScoreVersion != 2 || next.Notify[0].ChangeVersion != 21 {
		t.Fatal(next)
	}
	var stored model.Grade
	if err := db.First(&stored, "student_id = ? AND jxb_id = ?", "legacy", "unchanged").Error; err != nil || !reflect.DeepEqual(stored, unchanged) {
		t.Fatal(stored, err)
	}
	countRows(t, db, &model.GradeIdentityAlias{}, 2)
	countRows(t, db, &model.GradeNotificationOutbox{}, 3)
	countRows(t, db, &model.GradeSyncState{}, 1)
}

func TestLegacyActivationRollsBackAndRetriesWithBaseline(t *testing.T) {
	for _, retry := range []bool{false, true} {
		t.Run(fmt.Sprint(retry), func(t *testing.T) {
			db, d := newGradeTestDAO(t)
			legacy := model.Grade{StudentId: "legacy", JxbId: "class", Cj: 80, ChangeVersion: 7}
			if err := db.Create(&legacy).Error; err != nil {
				t.Fatal(err)
			}
			calls := 0
			if err := db.Callback().Create().Before("gorm:create").Register("test:activation-outbox", func(tx *gorm.DB) {
				if tx.Statement.Table == "grade_notification_outbox" {
					calls++
					if retry && calls == 1 {
						tx.AddError(&mysql.MySQLError{Number: 1213, Message: "deadlock"})
					} else if !retry {
						tx.AddError(errors.New("outbox unavailable"))
					}
				}
			}); err != nil {
				t.Fatal(err)
			}
			legacy.Cj = 90
			result, err := d.SyncGrades(context.Background(), snapshot("legacy", legacy))
			if retry {
				if err != nil || calls != 2 || len(result.Notify) != 1 || result.Notify[0].ScoreVersion != 1 || len(result.Changed) != 1 {
					t.Fatal(result, err, calls)
				}
				countRows(t, db, &model.GradeNotificationOutbox{}, 1)
				countRows(t, db, &model.GradeSyncState{}, 1)
				countRows(t, db, &model.GradeIdentityAlias{}, 1)
			} else {
				if err == nil || len(result.Changed) != 0 || len(result.Notify) != 0 {
					t.Fatal(result, err)
				}
				stored, err := d.FindGrades(context.Background(), "legacy", 0, 0)
				if err != nil || len(stored) != 1 || stored[0].NotificationID != nil || stored[0].ScoreVersion != 0 || stored[0].Cj != 80 || stored[0].ChangeVersion != 7 {
					t.Fatal(stored, err)
				}
				countRows(t, db, &model.GradeNotificationOutbox{}, 0)
				countRows(t, db, &model.GradeSyncState{}, 0)
				countRows(t, db, &model.GradeIdentityAlias{}, 0)
			}
		})
	}
}

func TestLegacyConcurrentFirstActivationHasOneIdentityAndEvent(t *testing.T) {
	db, d := newGradeTestDAO(t)
	if err := db.Create(&model.Grade{StudentId: "legacy", JxbId: "class", Cj: 70, ChangeVersion: 5}).Error; err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan GradeSyncResult, 2)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, jd := range []float32{2, 3} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			result, err := d.SyncGrades(context.Background(), snapshot("legacy", model.Grade{StudentId: "legacy", JxbId: "class", Cj: 80, Jd: jd}))
			results <- result
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	var id string
	notified := 0
	versions := []int64{}
	for range 2 {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
		result := <-results
		if len(result.Changed) != 1 || result.Changed[0].ScoreVersion != 1 {
			t.Fatal(result)
		}
		current := result.Changed[0].NotificationIdentity()
		if id != "" && id != current {
			t.Fatal("concurrent activation assigned different identities")
		}
		id = current
		versions = append(versions, result.Changed[0].ChangeVersion)
		notified += len(result.Notify)
	}
	slices.Sort(versions)
	if notified != 1 || len(id) != 32 || !slices.Equal(versions, []int64{6, 7}) {
		t.Fatal(notified, id, versions)
	}
	countRows(t, db, &model.GradeSyncState{}, 1)
	countRows(t, db, &model.GradeIdentityAlias{}, 1)
	countRows(t, db, &model.GradeNotificationOutbox{}, 1)
}

func TestNullLegacyScoreBecomesValidZero(t *testing.T) {
	db, d := newGradeTestDAO(t)
	if err := db.Exec("INSERT INTO grades (student_id,jxb_id,cj,change_version) VALUES ('legacy','class',NULL,7)").Error; err != nil {
		t.Fatal(err)
	}
	got := syncGrades(t, d, "legacy", model.Grade{StudentId: "legacy", JxbId: "class", Cj: 0})
	if len(got.Notify) != 1 || len(got.Changed) != 1 || got.Notify[0].ScoreVersion != 1 || got.Notify[0].ChangeVersion != 8 || got.Notify[0].NotificationID == nil {
		t.Fatal(got)
	}
	var row model.GradeNotificationOutbox
	if err := db.First(&row).Error; err != nil {
		t.Fatal(err)
	}
	event, err := domain.ParseGradeNotification(row)
	if err != nil || event.ExtendFields["change_reason"] != "first_published" {
		t.Fatal(event, err)
	}
	if again := syncGrades(t, d, "legacy", got.Notify[0]); len(again.Notify) != 0 || len(again.Changed) != 0 {
		t.Fatal(again)
	}
}

func TestUninitializedExistingStateDoesNotInheritLegacyBaseline(t *testing.T) {
	db, d := newGradeTestDAO(t)
	if err := db.Create(&model.Grade{StudentId: "bootstrap", JxbId: "class", Cj: 80, ChangeVersion: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.GradeSyncState{StudentID: "bootstrap"}).Error; err != nil {
		t.Fatal(err)
	}
	got := syncGrades(t, d, "bootstrap", model.Grade{StudentId: "bootstrap", JxbId: "class", Cj: 90})
	if len(got.Notify) != 0 || len(got.Changed) != 1 || got.Changed[0].ScoreVersion != 1 {
		t.Fatal(got)
	}
	countRows(t, db, &model.GradeNotificationOutbox{}, 0)
}

func TestLegacyPseudoIdentityIsNotRecreatedAsNewCourse(t *testing.T) {
	for _, renamed := range []bool{false, true} {
		t.Run(fmt.Sprint(renamed), func(t *testing.T) {
			db, d := newGradeTestDAO(t)
			old := model.Grade{StudentId: "legacy", JxbId: "课程20201", Kcmc: "课程", Xnm: 2020, Xqm: 1, Cj: 80, ChangeVersion: 3}
			if err := db.Create(&old).Error; err != nil {
				t.Fatal(err)
			}
			if same := syncGrades(t, d, "legacy", old); len(same.Changed) != 0 || len(same.Notify) != 0 {
				t.Fatal(same)
			}
			candidate := old
			candidate.JxbId = "real-class"
			if renamed {
				candidate.Kcmc = "课程也改名"
			}
			if result, err := d.SyncGrades(context.Background(), snapshot("legacy", candidate)); err != nil || len(result.Skipped) != 1 || len(result.Notify) != 0 {
				t.Fatal("unconfirmed old identity was treated as a new course", result, err)
			}
			countRows(t, db, &model.Grade{}, 1)
			countRows(t, db, &model.GradeSyncIssue{}, 1)
			countRows(t, db, &model.GradeNotificationOutbox{}, 0)
			countRows(t, db, &model.GradeIdentityAlias{}, 0)
			// 完整快照仍包含未变的旧伪 ID 时，不影响另一门明确的新课。
			fresh := model.Grade{StudentId: "legacy", JxbId: "another-course", Kcmc: "另一门课", Cj: 0}
			got := syncGrades(t, d, "legacy", old, fresh)
			if len(got.Notify) != 1 || got.Notify[0].JxbId != fresh.JxbId {
				t.Fatal(got)
			}
		})
	}
}

func TestInvalidNotificationStateIsQuarantinedOnTouch(t *testing.T) {
	for _, item := range []struct {
		id      *string
		version int64
	}{{nil, 1}, {stringPointer(""), 0}, {stringPointer("invalid"), 1}, {stringPointer("0123456789abcdef0123456789abcdef"), 0}} {
		db, d := newGradeTestDAO(t)
		if err := db.Create(&model.Grade{StudentId: "legacy", JxbId: "bad", Cj: 80, ChangeVersion: 1, NotificationID: item.id, ScoreVersion: item.version}).Error; err != nil {
			t.Fatal(err)
		}
		_, err := d.SyncGrades(context.Background(), snapshot("legacy", model.Grade{StudentId: "legacy", JxbId: "new", Cj: 90}, model.Grade{StudentId: "legacy", JxbId: "bad", Cj: 80}))
		if !errors.Is(err, ErrGradeIdentity) {
			t.Fatal(err)
		}
		countRows(t, db, &model.Grade{}, 1)
		countRows(t, db, &model.GradeSyncState{}, 0)
		countRows(t, db, &model.GradeIdentityAlias{}, 0)
		countRows(t, db, &model.GradeNotificationOutbox{}, 0)
		countRows(t, db, &model.GradeSyncIssue{}, 1)
	}
}

type legacyNotificationColumn struct {
	NotificationID string `gorm:"type:varchar(32);not null"`
}

func (legacyNotificationColumn) TableName() string { return "grades" }

func TestNotificationMigrationRelaxesOldConstraintWithoutReset(t *testing.T) {
	db, d := newGradeTestDAO(t)
	base := syncGrades(t, d, "student", model.Grade{StudentId: "student", JxbId: "class", Cj: 80}).Changed[0]
	base.Cj = 90
	base = syncGrades(t, d, "student", base).Notify[0]
	if err := db.Migrator().AlterColumn(&legacyNotificationColumn{}, "NotificationID"); err != nil {
		t.Fatal(err)
	}
	unique, err := hasGradeNotificationUniqueIndex(db)
	if err != nil {
		t.Fatal(err)
	}
	if !unique {
		if err := db.Migrator().CreateIndex(&model.Grade{}, gradeNotificationIdentityIndex); err != nil {
			t.Fatal(err)
		}
	}
	if err := InitTables(db); err == nil {
		t.Fatal("old NOT NULL identity accepted without explicit DDL")
	}
	for range 2 {
		if err := MigrateGradeNotifications(context.Background(), db); err != nil {
			t.Fatal(err)
		}
	}
	stored, err := d.FindGrades(context.Background(), "student", 0, 0)
	if err != nil || len(stored) != 1 || !reflect.DeepEqual(stored[0], base) {
		t.Fatal(stored, err)
	}
	countRows(t, db, &model.GradeIdentityAlias{}, 1)
	countRows(t, db, &model.GradeSyncState{}, 1)
	countRows(t, db, &model.GradeNotificationOutbox{}, 1)
	if unchanged := syncGrades(t, d, "student", base); len(unchanged.Notify) != 0 || len(unchanged.Changed) != 0 {
		t.Fatal(unchanged)
	}
}

func stringPointer(s string) *string { return &s }

func TestPartialGradeImportDoesNotBecomeLegacyBaseline(t *testing.T) {
	db, d := newGradeTestDAO(t)
	first := model.Grade{StudentId: "student", JxbId: "first", Cj: 80}
	partial := snapshot("student", first)
	partial.Complete = false
	for range 2 {
		if result, err := d.SyncGrades(context.Background(), partial); err != nil || len(result.Notify) != 0 {
			t.Fatal(result, err)
		}
		var state model.GradeSyncState
		if err := db.First(&state).Error; err != nil || state.InitializedAt != 0 {
			t.Fatal(state, err)
		}
	}
	second := model.Grade{StudentId: "student", JxbId: "second", Cj: 90}
	if result := syncGrades(t, d, "student", first, second); len(result.Notify) != 0 {
		t.Fatal(result)
	}
	countRows(t, db, &model.Grade{}, 2)
	countRows(t, db, &model.GradeNotificationOutbox{}, 0)
	first.Cj = 85
	partial.Grades = []model.Grade{first}
	if result, err := d.SyncGrades(context.Background(), partial); err != nil || len(result.Notify) != 1 || result.Notify[0].Cj != 85 {
		t.Fatal(result, err)
	}
	countRows(t, db, &model.GradeNotificationOutbox{}, 1)
}

func TestLegacyEmptyClassIDsStayUntrackedWithoutBlockingOtherCourses(t *testing.T) {
	db, d := newGradeTestDAO(t)
	old := model.Grade{StudentId: "legacy", JxbId: "课程20201", Kcmc: "课程", Xnm: 2020, Xqm: 1, Cj: 80, ChangeVersion: 3}
	if err := db.Create(&old).Error; err != nil {
		t.Fatal(err)
	}
	old.JxbId = ""
	fresh := model.Grade{StudentId: "legacy", JxbId: "real-class", Kcmc: "新课程", Cj: 90}
	for i, score := range []float32{80, 85, 85} {
		old.Cj = score
		got := syncGrades(t, d, "legacy", old, fresh)
		want := 0
		if i == 0 {
			want = 1
		}
		if len(got.Notify) != want || len(got.Skipped) != 0 {
			t.Fatal(got)
		}
		var stored model.Grade
		if err := db.First(&stored, "student_id = ? AND jxb_id = ?", "legacy", "课程20201").Error; err != nil || stored.Cj != score || stored.NotificationID != nil || stored.ScoreVersion != 0 {
			t.Fatal(stored, err)
		}
	}
	countRows(t, db, &model.Grade{}, 2)
	countRows(t, db, &model.GradeNotificationOutbox{}, 1)
	countRows(t, db, &model.GradeSyncIssue{}, 0)
}

func TestNewPseudoGradesOnlyUpdateLocalCache(t *testing.T) {
	db, d := newGradeTestDAO(t)
	syncGrades(t, d, "student")
	grade := model.Grade{StudentId: "student", Kcmc: "课程", Xnm: 2020, Xqm: 1, Cj: 80}
	for _, score := range []float32{80, 90} {
		grade.Cj = score
		got := syncGrades(t, d, "student", grade)
		if len(got.Changed) != 1 || len(got.Notify) != 0 || got.Changed[0].JxbId != "课程20201" || got.Changed[0].NotificationID != nil || got.Changed[0].ScoreVersion != 0 {
			t.Fatal(got)
		}
	}
	countRows(t, db, &model.Grade{}, 1)
	countRows(t, db, &model.GradeNotificationOutbox{}, 0)
	countRows(t, db, &model.GradeIdentityAlias{}, 0)
}

func TestAmbiguousPseudoGradesDoNotBlockConfirmedGradeChanges(t *testing.T) {
	db, d := newGradeTestDAO(t)
	real := model.Grade{StudentId: "student", JxbId: "real", Cj: 80}
	syncGrades(t, d, "student", real)
	first := model.Grade{StudentId: "student", Kcmc: "同名课程", Xnm: 2020, Xqm: 1, Cj: 70}
	legacy := first
	legacy.JxbId = legacy.LegacyJxbID()
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	second := first
	second.Cj = 90
	real.Cj = 85
	fresh := model.Grade{StudentId: "student", JxbId: "new-real", Cj: 90}
	got := syncGrades(t, d, "student", first, second, real, fresh)
	if len(got.Skipped) != 2 || len(got.Notify) != 2 || got.Notify[0].JxbId != "real" || got.Notify[1].JxbId != "new-real" {
		t.Fatal(got)
	}
	countRows(t, db, &model.Grade{}, 3)
	countRows(t, db, &model.GradeNotificationOutbox{}, 2)
	countRows(t, db, &model.GradeSyncIssue{}, 1)
}

func TestSuppressedNotificationsDoNotStopGradeSync(t *testing.T) {
	db, d := newGradeTestDAO(t)
	grade := model.Grade{StudentId: "student", JxbId: "real", Cj: 80}
	for _, score := range []float32{80, 90} {
		grade.Cj = score
		s := snapshot("student", grade)
		s.SuppressNotifications = true
		got, err := d.SyncGrades(context.Background(), s)
		if err != nil || len(got.Changed) != 1 || len(got.Notify) != 0 {
			t.Fatal(got, err)
		}
	}
	// 重新启用后首次完整导入保持静默，不补发关闭期间的历史变化。
	grade.Cj = 95
	if got := syncGrades(t, d, "student", grade); len(got.Notify) != 0 {
		t.Fatal(got)
	}
	countRows(t, db, &model.GradeNotificationOutbox{}, 0)
	grade.Cj = 100
	if got := syncGrades(t, d, "student", grade); len(got.Notify) != 1 {
		t.Fatal(got)
	}
	countRows(t, db, &model.GradeNotificationOutbox{}, 1)
}

func TestUnresolvedCourseDoesNotPreventKnownCourseBaseline(t *testing.T) {
	db, d := newGradeTestDAO(t)
	known := model.Grade{StudentId: "student", JxbId: "known", Cj: 80}
	unknown := model.Grade{StudentId: "student", Cj: 70}
	first := syncGrades(t, d, "student", known, unknown)
	if len(first.Notify) != 0 || len(first.Skipped) != 1 {
		t.Fatal(first)
	}
	known.Cj = 90
	next := syncGrades(t, d, "student", known, unknown)
	if len(next.Notify) != 1 || next.Notify[0].JxbId != "known" || len(next.Skipped) != 1 {
		t.Fatal(next)
	}
	countRows(t, db, &model.GradeNotificationOutbox{}, 1)
}

func TestLegacyBaselineIsInheritedWithoutPaginationTotal(t *testing.T) {
	db, d := newGradeTestDAO(t)
	old := model.Grade{StudentId: "legacy", JxbId: "old", Cj: 80, ChangeVersion: 3}
	if err := db.Create(&old).Error; err != nil {
		t.Fatal(err)
	}
	old.Cj = 90
	s := snapshot("legacy", old, model.Grade{StudentId: "legacy", JxbId: "new", Cj: 0})
	s.Complete = false
	got, err := d.SyncGrades(context.Background(), s)
	if err != nil || len(got.Notify) != 2 {
		t.Fatal(got, err)
	}
	var state model.GradeSyncState
	if err := db.First(&state).Error; err != nil || state.InitializedAt == 0 {
		t.Fatal(state, err)
	}
	countRows(t, db, &model.GradeNotificationOutbox{}, 2)
}

func TestUnscoredObservationsAreReadOnlyAndStudentBound(t *testing.T) {
	db, d := newGradeTestDAO(t)
	old := model.Grade{StudentId: "legacy", JxbId: "课程20201", Kcmc: "课程", Xnm: 2020, Xqm: 1, Cj: 80, ChangeVersion: 3}
	if err := db.Create(&old).Error; err != nil {
		t.Fatal(err)
	}
	observation := old
	observation.JxbId, observation.Cj = "", 999
	s := snapshot("legacy", model.Grade{StudentId: "legacy", JxbId: "new-real", Cj: 90})
	s.Unscored = []model.Grade{observation}
	got, err := d.SyncGrades(context.Background(), s)
	if err != nil || len(got.Notify) != 1 || len(got.Skipped) != 0 {
		t.Fatal(got, err)
	}
	var stored model.Grade
	if err := db.First(&stored, "student_id = ? AND jxb_id = ?", old.StudentId, old.JxbId).Error; err != nil || !reflect.DeepEqual(stored, old) {
		t.Fatal(stored, err)
	}
	s.Unscored[0].StudentId = "other-student"
	if _, err := d.SyncGrades(context.Background(), s); err == nil {
		t.Fatal("cross-student observations accepted")
	}
	countRows(t, db, &model.Grade{}, 2)
	countRows(t, db, &model.GradeNotificationOutbox{}, 1)
}
