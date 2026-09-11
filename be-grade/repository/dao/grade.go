package dao

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/asynccnu/ccnubox-be/be-grade/repository/model"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GradeDAO 数据库操作的集合
type GradeDAO interface {
	FirstOrCreate(ctx context.Context, grade *model.Grade) error
	FindGrades(ctx context.Context, studentId string, Xnm int64, Xqm int64) ([]model.Grade, error)
	BatchInsertOrUpdate(ctx context.Context, grades []model.Grade, ifDetail bool) (updateGrade []model.Grade, err error)
	GetDistinctGradeType(ctx context.Context, stuID string) ([]string, error)
}

type gradeDAO struct {
	db *gorm.DB
}

// NewGradeDAO 构建数据库操作实例
func NewGradeDAO(db *gorm.DB) GradeDAO {
	return &gradeDAO{db: db}
}

// FirstOrCreate 会自动查找是否存在记录,如果不存在则会存储
func (d *gradeDAO) FirstOrCreate(ctx context.Context, grade *model.Grade) error {
	err := d.db.WithContext(ctx).
		Where("student_id = ? AND jxb_id = ?", grade.StudentId, grade.JxbId).
		FirstOrCreate(grade).Error
	if err != nil {
		return errorx.Errorf("dao: FirstOrCreate failed, sid: %s, jxb: %s, err: %w", grade.StudentId, grade.JxbId, err)
	}
	return nil
}

// FindGrades 搜索成绩,xnm(学年名),xqm(学期名)条件为可选
func (d *gradeDAO) FindGrades(ctx context.Context, studentId string, Xnm int64, Xqm int64) ([]model.Grade, error) {
	var grades []model.Grade

	query := d.db.WithContext(ctx).Model(&model.Grade{}).Where("student_id = ?", studentId)
	if Xnm != 0 {
		query = query.Where("xnm = ?", Xnm)
	}
	if Xqm != 0 {
		query = query.Where("xqm = ?", Xqm)
	}

	err := query.Find(&grades).Error
	if err != nil {
		return nil, errorx.Errorf("dao: FindGrades failed, sid: %s, xnm: %d, xqm: %d, err: %w", studentId, Xnm, Xqm, err)
	}

	return grades, nil
}

const maxGradeTransactionAttempts = 5

// BatchInsertOrUpdate 批量处理成绩同步逻辑
func (d *gradeDAO) BatchInsertOrUpdate(ctx context.Context, grades []model.Grade, ifDetail bool) ([]model.Grade, error) {
	if len(grades) == 0 {
		return nil, nil
	}

	values := make([][]interface{}, 0, len(grades))
	// 构造联合键并规格化 ID
	for i := range grades {
		grades[i].JxbId = normalizeJxbId(&grades[i])
		values = append(values, []interface{}{grades[i].StudentId, grades[i].JxbId})
	}

	var (
		affectedGrades []model.Grade
		err            error
	)
	for attempt := 1; attempt <= maxGradeTransactionAttempts; attempt++ {
		affectedGrades = nil
		err = d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			// 行锁持有到事务提交，确保同一成绩的版本读取和递增不会交错。
			var existingGrades []model.Grade
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("(student_id, jxb_id) IN ?", values).
				Find(&existingGrades).Error; err != nil {
				return errorx.Errorf("find existing records failed: %w", err)
			}

			existingMap := make(map[string]model.Grade, len(existingGrades))
			for _, grade := range existingGrades {
				key := grade.StudentId + grade.JxbId
				existingMap[key] = grade
			}

			toInsert := make([]model.Grade, 0, len(grades))
			toUpdate := make([]model.Grade, 0, len(grades))
			for _, grade := range grades {
				key := grade.StudentId + grade.JxbId
				existing, exists := existingMap[key]
				if !exists {
					grade.ChangeVersion = 1
					toInsert = append(toInsert, grade)
					continue
				}
				if !isGradeEqual(existing, grade, ifDetail) {
					grade.ChangeVersion = existing.ChangeVersion + 1
					toUpdate = append(toUpdate, grade)
				}
			}

			// 不对新增行执行覆盖式 upsert。并发插入冲突会回滚并重试，重新读取已提交行后再分配版本。
			if len(toInsert) > 0 {
				if err := tx.Create(&toInsert).Error; err != nil {
					return errorx.Errorf("bulk insert failed, count: %d: %w", len(toInsert), err)
				}
			}
			if len(toUpdate) > 0 {
				if err := tx.Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "student_id"}, {Name: "jxb_id"}},
					DoUpdates: clause.AssignmentColumns(gradeUpdateColumns(ifDetail)),
				}).Create(&toUpdate).Error; err != nil {
					return errorx.Errorf("bulk upsert failed, count: %d: %w", len(toUpdate), err)
				}
			}

			affectedGrades = make([]model.Grade, 0, len(toInsert)+len(toUpdate))
			affectedGrades = append(affectedGrades, toInsert...)
			affectedGrades = append(affectedGrades, toUpdate...)
			return nil
		})
		if err == nil {
			return affectedGrades, nil
		}
		if ctx.Err() != nil || !isRetryableGradeTransactionError(err) || attempt == maxGradeTransactionAttempts {
			break
		}
		time.Sleep(time.Duration(attempt) * 10 * time.Millisecond)
	}

	return nil, errorx.Errorf("dao: BatchInsertOrUpdate transaction failed, count: %d, err: %w", len(grades), err)
}

func isRetryableGradeTransactionError(err error) bool {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		switch mysqlErr.Number {
		case 1062, 1205, 1213:
			return true
		}
	}

	// SQLite 仅用于 DAO 测试，其并发写锁同样需要重试整个事务。
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "database is locked") || strings.Contains(message, "database table is locked")
}

func gradeUpdateColumns(ifDetail bool) []string {
	columns := []string{
		"kc_id", "kcmc", "xnm", "xqm", "xf", "kcxzmc", "kclbmc", "kcbj", "jd", "cj", "change_version",
	}
	if ifDetail {
		columns = append(columns,
			"regular_grade_percent", "regular_grade", "final_grade_percent", "final_grade",
		)
	}
	return columns
}

func (d *gradeDAO) GetDistinctGradeType(ctx context.Context, stuID string) ([]string, error) {
	var results []string
	err := d.db.WithContext(ctx).Model(&model.Grade{}).
		Where("student_id = ?", stuID).
		Distinct("kcxzmc").
		Pluck("kcxzmc", &results).Error
	if err != nil {
		return nil, errorx.Errorf("dao: GetDistinctGradeType failed, sid: %s, err: %w", stuID, err)
	}
	return results, nil
}

// 内部辅助函数
func normalizeJxbId(g *model.Grade) string {
	if g.JxbId != "" {
		return g.JxbId
	}
	// 兜底逻辑：通过课程名+学年学期生成伪 ID
	return g.Kcmc + strconv.FormatInt(g.Xnm, 10) + strconv.FormatInt(g.Xqm, 10)
}

func isGradeEqual(a, b model.Grade, ifDetail bool) bool {
	// 基础比较字段
	baseEqual := a.Kcmc == b.Kcmc &&
		a.KcId == b.KcId &&
		a.Xnm == b.Xnm &&
		a.Xqm == b.Xqm &&
		a.Xf == b.Xf &&
		a.Kcxzmc == b.Kcxzmc &&
		a.Kclbmc == b.Kclbmc &&
		a.Kcbj == b.Kcbj &&
		a.Jd == b.Jd &&
		a.Cj == b.Cj

	if !ifDetail {
		return baseEqual
	}

	// 详情比较字段
	return baseEqual &&
		a.RegularGradePercent == b.RegularGradePercent &&
		a.RegularGrade == b.RegularGrade &&
		a.FinalGradePercent == b.FinalGradePercent &&
		a.FinalGrade == b.FinalGrade
}
