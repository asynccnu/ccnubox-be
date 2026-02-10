package dao

import (
	"context"
	"strconv"

	"github.com/asynccnu/ccnubox-be/be-grade/repository/model"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"gorm.io/gorm"
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

// BatchInsertOrUpdate 批量处理成绩同步逻辑
func (d *gradeDAO) BatchInsertOrUpdate(ctx context.Context, grades []model.Grade, ifDetail bool) (affectedGrades []model.Grade, err error) {
	if len(grades) == 0 {
		return nil, nil
	}

	// 构造联合键并规格化 ID
	ids := make([]string, len(grades))
	for i := range grades {
		grades[i].JxbId = normalizeJxbId(&grades[i])
		ids[i] = grades[i].StudentId + grades[i].JxbId
	}

	// 1. 查询已有记录用于比对
	var existingGrades []model.Grade
	err = d.db.WithContext(ctx).
		Where("CONCAT(student_id, jxb_id) IN ?", ids).
		Find(&existingGrades).Error
	if err != nil {
		return nil, errorx.Errorf("dao: BatchInsertOrUpdate find existing records failed, count: %d, err: %w", len(ids), err)
	}

	existingMap := make(map[string]model.Grade)
	for _, grade := range existingGrades {
		key := grade.StudentId + grade.JxbId
		existingMap[key] = grade
	}

	var toInsert []model.Grade
	var toUpdate []model.Grade

	for _, grade := range grades {
		key := grade.StudentId + grade.JxbId
		if existing, exists := existingMap[key]; !exists {
			toInsert = append(toInsert, grade)
		} else {
			// 比对字段是否有变化
			if !isGradeEqual(existing, grade, ifDetail) {
				toUpdate = append(toUpdate, grade)
			}
		}
	}

	// 2. 执行插入
	if len(toInsert) > 0 {
		if err = d.db.WithContext(ctx).Create(&toInsert).Error; err != nil {
			return nil, errorx.Errorf("dao: BatchInsertOrUpdate bulk insert failed, count: %d, err: %w", len(toInsert), err)
		}
	}

	// 3. 执行更新
	if len(toUpdate) > 0 {
		// 这里使用事务保证批量更新的一致性
		err = d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			for _, g := range toUpdate {
				if err := tx.Model(&model.Grade{}).
					Where("student_id = ? AND jxb_id = ?", g.StudentId, g.JxbId).
					Updates(&g).Error; err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return nil, errorx.Errorf("dao: BatchInsertOrUpdate bulk update transaction failed, count: %d, err: %w", len(toUpdate), err)
		}
	}

	affectedGrades = append(toInsert, toUpdate...)
	return affectedGrades, nil
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
