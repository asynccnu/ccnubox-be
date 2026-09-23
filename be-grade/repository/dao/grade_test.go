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
	results := make(chan []model.Grade, len(updates))
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
	for changed := range results {
		if len(changed) != 1 {
			t.Errorf("BatchInsertOrUpdate() changed count = %d, want 1", len(changed))
			continue
		}
		versions = append(versions, changed[0].ChangeVersion)
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
	if _, err = d.BatchInsertOrUpdate(context.Background(), []model.Grade{detail}, true); err != nil {
		t.Fatal(err)
	}
	// 同一详情再次投递应无写入，即使其携带的版本已经落后。
	if changed, err := d.BatchInsertOrUpdate(context.Background(), []model.Grade{detail}, true); err != nil || len(changed) != 0 {
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
	if changed, err := d.BatchInsertOrUpdate(context.Background(), []model.Grade{detail}, true); err != nil || len(changed) != 0 {
		t.Fatalf("resurrected deleted grade: changed=%v err=%v", changed, err)
	}
}
