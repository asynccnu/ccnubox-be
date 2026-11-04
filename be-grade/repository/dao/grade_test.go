package dao

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/asynccnu/ccnubox-be/be-grade/repository/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestGradeUpdateColumns(t *testing.T) {
	baseWant := []string{
		"kc_id", "kcmc", "xnm", "xqm", "xf", "kcxzmc", "kclbmc", "kcbj", "jd", "cj", "change_version",
	}
	base := gradeUpdateColumns(false)
	if !slices.Equal(base, baseWant) {
		t.Fatalf("gradeUpdateColumns(false) = %v, want %v", base, baseWant)
	}

	detailWant := []string{"regular_grade_percent", "regular_grade", "final_grade_percent", "final_grade", "change_version"}
	detail := gradeUpdateColumns(true)
	if !slices.Equal(detail, detailWant) {
		t.Fatalf("gradeUpdateColumns(true) = %v, want %v", detail, detailWant)
	}
}

func TestBatchInsertOrUpdateAllocatesConcurrentChangeVersions(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "grade.db") + "?_journal_mode=WAL&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err = db.AutoMigrate(&model.Grade{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}

	initial := model.Grade{
		StudentId:     "student-id",
		JxbId:         "class-id",
		Kcmc:          "课程",
		Cj:            70,
		Jd:            1,
		ChangeVersion: 1,
	}
	if err = db.Create(&initial).Error; err != nil {
		t.Fatalf("insert initial grade: %v", err)
	}

	dao := NewGradeDAO(db)
	updates := []model.Grade{
		{StudentId: initial.StudentId, JxbId: initial.JxbId, Kcmc: initial.Kcmc, Cj: 80, Jd: 2},
		{StudentId: initial.StudentId, JxbId: initial.JxbId, Kcmc: initial.Kcmc, Cj: 80, Jd: 3},
	}
	start := make(chan struct{})
	results := make(chan GradeSyncResult, len(updates))
	errs := make(chan error, len(updates))
	var wg sync.WaitGroup
	for i := range updates {
		wg.Add(1)
		go func(grade model.Grade) {
			defer wg.Done()
			<-start
			changed, err := dao.BatchInsertOrUpdate(context.Background(), []model.Grade{grade}, false)
			if err != nil {
				errs <- err
				return
			}
			results <- changed
		}(updates[i])
	}
	close(start)
	wg.Wait()
	close(results)
	close(errs)

	for err := range errs {
		t.Errorf("BatchInsertOrUpdate() error = %v", err)
	}
	versions := make([]int64, 0, len(updates))
	notifications := 0
	for result := range results {
		if len(result.Changed) != 1 {
			t.Errorf("BatchInsertOrUpdate() changed count = %d, want 1", len(result.Changed))
			continue
		}
		versions = append(versions, result.Changed[0].ChangeVersion)
		notifications += len(result.Notify)
	}
	if notifications != 1 {
		t.Fatalf("notification count = %d, want 1 total score change", notifications)
	}
	slices.Sort(versions)
	if want := []int64{2, 3}; !slices.Equal(versions, want) {
		t.Fatalf("change versions = %v, want %v", versions, want)
	}

	var final model.Grade
	if err = db.Where("student_id = ? AND jxb_id = ?", initial.StudentId, initial.JxbId).First(&final).Error; err != nil {
		t.Fatalf("find final grade: %v", err)
	}
	if final.ChangeVersion != 3 {
		t.Fatalf("final change version = %d, want 3", final.ChangeVersion)
	}
}

func TestDetailUpdateRejectsStaleVersionAndNeverRewritesBase(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "details.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&model.Grade{}); err != nil {
		t.Fatal(err)
	}
	initial := model.Grade{StudentId: "student", JxbId: "class", Cj: 80, ChangeVersion: 2, RegularGradePercent: "未知", FinalGradePercent: "未知"}
	if err = db.Create(&initial).Error; err != nil {
		t.Fatal(err)
	}
	d := NewGradeDAO(db)
	detail := initial
	detail.Cj = 20
	detail.ChangeVersion = 1
	detail.RegularGradePercent = "30"
	detail.FinalGradePercent = "70"
	detail.RegularGrade = 80
	detail.FinalGrade = 80
	if _, err = d.BatchInsertOrUpdate(context.Background(), []model.Grade{detail}, true); !errors.Is(err, ErrGradeVersionChanged) {
		t.Fatalf("stale update: %v", err)
	}
	detail.ChangeVersion = 2
	if result, err := d.BatchInsertOrUpdate(context.Background(), []model.Grade{detail}, true); err != nil || len(result.Changed) != 1 || len(result.Notify) != 0 || result.Changed[0].Cj != initial.Cj {
		t.Fatalf("detail update: result=%+v err=%v", result, err)
	}
	// 同一详情再次投递应无写入，即使其携带的版本已经落后。
	if changed, err := d.BatchInsertOrUpdate(context.Background(), []model.Grade{detail}, true); err != nil || len(changed.Changed) != 0 || len(changed.Notify) != 0 {
		t.Fatalf("duplicate detail: changed=%v err=%v", changed, err)
	}
	var got model.Grade
	if err = db.First(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.Cj != 80 || got.RegularGradePercent != "30" || got.ChangeVersion != 3 {
		t.Fatalf("unexpected grade after detail: %+v", got)
	}
	if err = db.Delete(&got).Error; err != nil {
		t.Fatal(err)
	}
	if changed, err := d.BatchInsertOrUpdate(context.Background(), []model.Grade{detail}, true); err != nil || len(changed.Changed) != 0 || len(changed.Notify) != 0 {
		t.Fatalf("resurrected deleted grade: changed=%v err=%v", changed, err)
	}
}

func newGradeTestDAO(t *testing.T) (*gorm.DB, GradeDAO) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "grades.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&model.Grade{}); err != nil {
		t.Fatal(err)
	}
	return db, NewGradeDAO(db)
}

func TestBatchInsertOrUpdateSeparatesStorageAndNotifications(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*model.Grade)
	}{
		{"name", func(g *model.Grade) { g.Kcmc = "新课程名" }},
		{"detail record id", func(g *model.Grade) { g.KcId = "new-record-id" }},
		{"year", func(g *model.Grade) { g.Xnm++ }},
		{"term", func(g *model.Grade) { g.Xqm++ }},
		{"credit", func(g *model.Grade) { g.Xf++ }},
		{"nature", func(g *model.Grade) { g.Kcxzmc = "选修" }},
		{"category", func(g *model.Grade) { g.Kclbmc = "公共课" }},
		{"mark", func(g *model.Grade) { g.Kcbj = "辅修" }},
		{"grade point", func(g *model.Grade) { g.Jd++ }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, d := newGradeTestDAO(t)
			initial := model.Grade{
				StudentId: "student", JxbId: "class", Kcmc: "原课程名", Cj: 80, ChangeVersion: 8,
				RegularGradePercent: "30", RegularGrade: 75, FinalGradePercent: "70", FinalGrade: 85,
			}
			if err := db.Create(&initial).Error; err != nil {
				t.Fatal(err)
			}
			incoming := initial
			tc.change(&incoming)
			incoming.RegularGradePercent = "未知"
			incoming.FinalGradePercent = "未知"
			incoming.RegularGrade = 0
			incoming.FinalGrade = 0
			result, err := d.BatchInsertOrUpdate(context.Background(), []model.Grade{incoming}, false)
			if err != nil || len(result.Changed) != 1 || len(result.Notify) != 0 {
				t.Fatalf("metadata update: result=%+v err=%v", result, err)
			}
			var saved model.Grade
			if err = db.First(&saved).Error; err != nil {
				t.Fatal(err)
			}
			if !isGradeEqual(saved, incoming, false) || !gradeDetailsEqual(saved, initial) || saved.ChangeVersion != 9 || result.Changed[0] != saved {
				t.Fatalf("unexpected saved record or result: saved=%+v result=%+v", saved, result)
			}
			if result, err = d.BatchInsertOrUpdate(context.Background(), []model.Grade{incoming}, false); err != nil || len(result.Changed) != 0 || len(result.Notify) != 0 {
				t.Fatalf("duplicate: result=%+v err=%v", result, err)
			}
		})
	}
}

func TestBatchInsertOrUpdateNotifiesEveryScoreChangeIncludingZeroAndOldTerms(t *testing.T) {
	_, d := newGradeTestDAO(t)
	grade := model.Grade{StudentId: "student", JxbId: "class", Xnm: 2020, Xqm: 1}
	for i, score := range []float32{0, 80, 90, 80, 0} {
		grade.Cj = score
		result, err := d.BatchInsertOrUpdate(context.Background(), []model.Grade{grade}, false)
		if err != nil || len(result.Changed) != 1 || len(result.Notify) != 1 {
			t.Fatalf("score %v: result=%+v err=%v", score, result, err)
		}
		if result.Notify[0] != result.Changed[0] || result.Notify[0].ChangeVersion != int64(i+1) {
			t.Fatalf("score %v: result=%+v", score, result)
		}
		if result, err = d.BatchInsertOrUpdate(context.Background(), []model.Grade{grade}, false); err != nil || len(result.Changed) != 0 || len(result.Notify) != 0 {
			t.Fatalf("duplicate score %v: result=%+v err=%v", score, result, err)
		}
	}
}

func TestBatchInsertOrUpdateMergesDuplicatesAndRejectsConflicts(t *testing.T) {
	db, d := newGradeTestDAO(t)
	grade := model.Grade{StudentId: "student", Kcmc: "课程", Xnm: 2025, Xqm: 1, Cj: 80}
	normalized := grade
	normalized.JxbId = normalizeJxbId(&grade)
	result, err := d.BatchInsertOrUpdate(context.Background(), []model.Grade{grade, normalized, grade}, false)
	if err != nil || len(result.Changed) != 1 || len(result.Notify) != 1 || result.Changed[0].ChangeVersion != 1 {
		t.Fatalf("duplicate insert: result=%+v err=%v", result, err)
	}
	if grade.JxbId != "" {
		t.Fatal("input grade was modified")
	}
	conflict := grade
	conflict.Cj = 90
	if result, err = d.BatchInsertOrUpdate(context.Background(), []model.Grade{grade, conflict}, false); err == nil || len(result.Changed) != 0 || len(result.Notify) != 0 {
		t.Fatalf("conflicting batch: result=%+v err=%v", result, err)
	}
	var saved model.Grade
	if err = db.First(&saved).Error; err != nil || saved.ChangeVersion != 1 || saved.Cj != 80 {
		t.Fatalf("conflicting batch wrote data: saved=%+v err=%v", saved, err)
	}
	grade.Cj = 90
	if result, err = d.BatchInsertOrUpdate(context.Background(), []model.Grade{grade, grade}, false); err != nil || len(result.Notify) != 1 || result.Notify[0].ChangeVersion != 2 {
		t.Fatalf("duplicate update: result=%+v err=%v", result, err)
	}
}

func TestBatchInsertOrUpdateDoesNotConcatenateIdentityKeys(t *testing.T) {
	db, d := newGradeTestDAO(t)
	initial := []model.Grade{
		{StudentId: "a", JxbId: "bc", Cj: 70, ChangeVersion: 2},
		{StudentId: "ab", JxbId: "c", Cj: 80, ChangeVersion: 5},
	}
	if err := db.Create(&initial).Error; err != nil {
		t.Fatal(err)
	}
	initial[0].Cj = 90
	result, err := d.BatchInsertOrUpdate(context.Background(), initial, false)
	if err != nil || len(result.Changed) != 1 || len(result.Notify) != 1 || result.Notify[0].StudentId != "a" || result.Notify[0].ChangeVersion != 3 {
		t.Fatalf("identity collision: result=%+v err=%v", result, err)
	}
}

func TestBatchInsertOrUpdateRollsBackAllResults(t *testing.T) {
	db, d := newGradeTestDAO(t)
	initial := model.Grade{StudentId: "student", JxbId: "existing", Cj: 80, ChangeVersion: 3}
	if err := db.Create(&initial).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TRIGGER reject_grade_update BEFORE UPDATE ON grades BEGIN SELECT RAISE(ABORT, 'test rollback'); END;").Error; err != nil {
		t.Fatal(err)
	}
	initial.Cj = 90
	result, err := d.BatchInsertOrUpdate(context.Background(), []model.Grade{
		{StudentId: "student", JxbId: "new", Cj: 70}, initial,
	}, false)
	if err == nil || len(result.Changed) != 0 || len(result.Notify) != 0 {
		t.Fatalf("failed transaction leaked results: result=%+v err=%v", result, err)
	}
	grades, err := d.FindGrades(context.Background(), "student", 0, 0)
	if err != nil || len(grades) != 1 || grades[0].Cj != 80 || grades[0].ChangeVersion != 3 {
		t.Fatalf("failed transaction wrote grades: grades=%+v err=%v", grades, err)
	}
}

func TestBatchInsertOrUpdateRebuildsResultsOnRetry(t *testing.T) {
	db, d := newGradeTestDAO(t)
	attempts := 0
	if err := db.Callback().Create().Before("gorm:create").Register("test:retry_grade_once", func(tx *gorm.DB) {
		attempts++
		if attempts == 1 {
			tx.AddError(errors.New("database is locked"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	result, err := d.BatchInsertOrUpdate(context.Background(), []model.Grade{{StudentId: "student", JxbId: "class", Cj: 0}}, false)
	if err != nil || attempts != 2 || len(result.Changed) != 1 || len(result.Notify) != 1 || result.Notify[0].ChangeVersion != 1 {
		t.Fatalf("retry leaked results or versions: result=%+v err=%v attempts=%d", result, err, attempts)
	}
}
