package dao

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-grade/repository/model"
	"gorm.io/gorm"
)

// GradeDAO 数据库操作的集合
type GradeDAO interface {
	FirstOrCreate(ctx context.Context, grade *model.Grade) error
	FindGrades(ctx context.Context, studentId string, Xnm int64, Xqm int64) ([]model.Grade, error)
	BatchInsertOrUpdate(ctx context.Context, grades []model.Grade, ifDetail bool) (updateGrade []model.Grade, err error)
}

type gradeDAO struct {
	db *gorm.DB
}

// NewDatabaseStruct  构建数据库操作实例
func NewGradeDAO(db *gorm.DB) GradeDAO {
	return &gradeDAO{db: db}
}

// FirstOrCreate 会自动查找是否存在记录,如果不存在则会存储
func (d *gradeDAO) FirstOrCreate(ctx context.Context, grade *model.Grade) error {
	return d.db.WithContext(ctx).Where("student_id = ? AND jxb_id = ?", grade.StudentId, grade.JxbId).FirstOrCreate(grade).Error
}

// FindAllGradesByStudentId 搜索成绩,xnm(学年名),xqm(学期名)条件为可选
func (d *gradeDAO) FindGrades(ctx context.Context, studentId string, Xnm int64, Xqm int64) ([]model.Grade, error) {
	// 定义查询结果的容器
	var grades []model.Grade

	// 构建查询
	query := d.db.WithContext(ctx).Model(&model.Grade{}).Where("student_id = ?", studentId)
	if Xnm != 0 { // 如果 Xnm 有值，拼接学年条件
		query = query.Where("xnm = ?", Xnm)
	}

	if Xqm != 0 { // 如果 Xqm 有值，拼接学期条件
		query = query.Where("xqm = ?", Xqm)
	}

	// 执行查询
	err := query.Find(&grades).Error
	if err != nil {
		return nil, err
	}

	return grades, nil
}

func (d *gradeDAO) BatchInsertOrUpdate(ctx context.Context, grades []model.Grade, ifDetail bool) (affectedGrades []model.Grade, err error) {

	// 构造联合键：student_id + jxb_id
	ids := make([]string, len(grades))
	for i, grade := range grades {
		ids[i] = grade.StudentId + grade.JxbId
	}

	// 查询已有记录
	var existingGrades []model.Grade
	if err = d.db.WithContext(ctx).
		Where("CONCAT(student_id, jxb_id) IN ?", ids).
		Find(&existingGrades).Error; err != nil {
		return nil, err
	}

	// 建立现有记录的Map方便比对
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
			// 你可以根据实际字段进行更精细的字段比较
			if !isGradeEqual(existing, grade, ifDetail) {
				toUpdate = append(toUpdate, grade)
			}
		}
	}

	// 插入新增记录
	if len(toInsert) > 0 {
		if err = d.db.WithContext(ctx).Create(&toInsert).Error; err != nil {
			return nil, err
		}
	}

	// 批量更新已有但内容有变化的记录
	if len(toUpdate) > 0 {
		for _, g := range toUpdate {
			if err = d.db.WithContext(ctx).
				Model(&model.Grade{}).
				Where("student_id = ? AND jxb_id = ?", g.StudentId, g.JxbId).
				Updates(&g).Error; err != nil {
				return nil, err
			}
		}
	}

	// 返回受影响的记录（新增 + 更新）
	affectedGrades = append(toInsert, toUpdate...)
	return affectedGrades, nil
}

func isGradeEqual(a, b model.Grade, ifDetail bool) bool {
	if ifDetail {
		return a.Kcmc == b.Kcmc &&
			a.Xnm == b.Xnm &&
			a.Xqm == b.Xqm &&
			a.Xf == b.Xf &&
			a.Kcxzmc == b.Kcxzmc &&
			a.Kclbmc == b.Kclbmc &&
			a.Kcbj == b.Kcbj &&
			a.Jd == b.Jd &&
			a.Cj == b.Cj &&
			a.RegularGradePercent == b.RegularGradePercent &&
			a.RegularGrade == b.RegularGrade &&
			a.FinalGradePercent == b.FinalGradePercent &&
			a.FinalGrade == b.FinalGrade
	}
	return a.Kcmc == b.Kcmc &&
		a.Xnm == b.Xnm &&
		a.Xqm == b.Xqm &&
		a.Xf == b.Xf &&
		a.Kcxzmc == b.Kcxzmc &&
		a.Kclbmc == b.Kclbmc &&
		a.Kcbj == b.Kcbj &&
		a.Jd == b.Jd &&
		a.Cj == b.Cj

	// 这里去除了对平时分的校验,因为成本太高了,但是事实上也可能会有变化,目前暂时不打算处理
	//a.RegularGradePercent == b.RegularGradePercent &&
	//a.RegularGrade == b.RegularGrade &&
	//a.FinalGradePercent == b.FinalGradePercent &&
	//a.FinalGrade == b.FinalGrade &&
}
