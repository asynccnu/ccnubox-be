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
	BatchInsertOrUpdate(ctx context.Context, grades []model.Grade) (updateGrade []model.Grade, err error)
	FindGraduateGrades(ctx context.Context, studentId string, year string, term int64) ([]model.GraduateGrade, error)
	BatchInsertOrUpdateGraduate(ctx context.Context, grades []model.GraduateGrade) (affectedGrades []model.GraduateGrade, err error)
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
	return d.db.WithContext(ctx).Where("student_id = ? AND jxb_id = ?", grade.Studentid, grade.JxbId).FirstOrCreate(grade).Error
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

func (d *gradeDAO) BatchInsertOrUpdate(ctx context.Context, grades []model.Grade) (affectedGrades []model.Grade, err error) {

	// 构造联合键：student_id + jxb_id
	ids := make([]string, len(grades))
	for i, grade := range grades {
		ids[i] = grade.Studentid + grade.JxbId
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
		key := grade.Studentid + grade.JxbId
		existingMap[key] = grade
	}

	var toInsert []model.Grade
	var toUpdate []model.Grade

	for _, grade := range grades {
		key := grade.Studentid + grade.JxbId

		if existing, exists := existingMap[key]; !exists {
			toInsert = append(toInsert, grade)
		} else {
			// 你可以根据实际字段进行更精细的字段比较
			if !isGradeEqual(existing, grade) {
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
			if err = d.db.WithContext(ctx).Save(&g).Error; err != nil {
				return nil, err
			}
		}
	}

	// 返回受影响的记录（新增 + 更新）
	affectedGrades = append(toInsert, toUpdate...)
	return affectedGrades, nil
}

func isGradeEqual(a, b model.Grade) bool {
	return a.Kcmc == b.Kcmc &&
		a.Xnm == b.Xnm &&
		a.Xqm == b.Xqm &&
		a.Xf == b.Xf &&
		a.Kcxzmc == b.Kcxzmc &&
		a.Kclbmc == b.Kclbmc &&
		a.Kcbj == b.Kcbj &&
		a.Jd == b.Jd &&
		a.RegularGradePercent == b.RegularGradePercent &&
		a.RegularGrade == b.RegularGrade &&
		a.FinalGradePercent == b.FinalGradePercent &&
		a.FinalGrade == b.FinalGrade &&
		a.Cj == b.Cj
}

func (d *gradeDAO) FindGraduateGrades(ctx context.Context, studentId string, year string, term int64) ([]model.GraduateGrade, error) {
	var grades []model.GraduateGrade

	query := d.db.WithContext(ctx).Model(&model.GraduateGrade{}).Where("student_id = ?", studentId)
	if year != "" {
		query = query.Where("year = ?", year)
	}
	if term != 0 {
		query = query.Where("term = ?", term)
	}

	if err := query.Find(&grades).Error; err != nil {
		return nil, err
	}
	return grades, nil
}

func (d *gradeDAO) BatchInsertOrUpdateGraduate(ctx context.Context, grades []model.GraduateGrade) (affectedGrades []model.GraduateGrade, err error) {
	if len(grades) == 0 {
		return nil, nil
	}

	ids := make([]string, len(grades))
	for i, g := range grades {
		ids[i] = g.StudentID + g.JxbId
	}

	var existing []model.GraduateGrade
	if err = d.db.WithContext(ctx).
		Where("CONCAT(student_id, jxb_id) IN ?", ids).
		Find(&existing).Error; err != nil {
		return nil, err
	}

	existMap := make(map[string]model.GraduateGrade, len(existing))
	for _, eg := range existing {
		existMap[eg.StudentID+eg.JxbId] = eg
	}

	var toInsert []model.GraduateGrade
	var toUpdate []model.GraduateGrade

	for _, g := range grades {
		key := g.StudentID + g.JxbId
		if old, ok := existMap[key]; !ok {
			toInsert = append(toInsert, g)
		} else if !isGraduateGradeEqual(old, g) {
			toUpdate = append(toUpdate, g)
		}
	}

	if len(toInsert) > 0 {
		if err = d.db.WithContext(ctx).Create(&toInsert).Error; err != nil {
			return nil, err
		}
	}

	if len(toUpdate) > 0 {
		for _, g := range toUpdate {
			if err = d.db.WithContext(ctx).Save(&g).Error; err != nil {
				return nil, err
			}
		}
	}

	affectedGrades = append(toInsert, toUpdate...)
	return affectedGrades, nil
}

func isGraduateGradeEqual(a, b model.GraduateGrade) bool {
	return a.Status == b.Status &&
		a.Year == b.Year &&
		a.Term == b.Term &&
		a.StudentID == b.StudentID &&
		a.Name == b.Name &&
		a.StudentCategory == b.StudentCategory &&
		a.College == b.College &&
		a.Major == b.Major &&
		a.Grade == b.Grade &&
		a.ClassCode == b.ClassCode &&
		a.ClassName == b.ClassName &&
		a.ClassNature == b.ClassNature &&
		a.Credit == b.Credit &&
		a.Point == b.Point &&
		a.GradePoints == b.GradePoints &&
		a.IsAvailable == b.IsAvailable &&
		a.IsDegree == b.IsDegree &&
		a.SetCollege == b.SetCollege &&
		a.ClassMark == b.ClassMark &&
		a.ClassCategory == b.ClassCategory &&
		a.ClassID == b.ClassID &&
		a.Teacher == b.Teacher
}
