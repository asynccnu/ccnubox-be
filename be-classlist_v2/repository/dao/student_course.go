package dao

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-classlist_v2/biz/errcode"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/repository/model"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"gorm.io/gorm/clause"
)

type StudentCourseDAO struct {
	BaseDAO
}

func NewStudentCourseDAO(base BaseDAO) StudentCourseDAO {
	return StudentCourseDAO{
		BaseDAO: base,
	}
}

func (s StudentCourseDAO) GetClassMetaData(ctx context.Context, stuID, year, semester string, claIds []string) map[string]model.ClassMetaData {
	logh := logger.From(ctx)

	// 初始化返回的 map
	res := make(map[string]model.ClassMetaData)
	if len(claIds) == 0 {
		return res
	}

	// 定义一个内部结构体用于接收扫描结果，因为需要 cla_id 作为 map 的 key
	type queryResult struct {
		ClaID           string `gorm:"column:cla_id"`
		IsManuallyAdded bool   `gorm:"column:is_manually_added"`
		Note            string `gorm:"column:note"`
	}

	var results []queryResult

	// 执行数据库查询
	// 约定将事务 db 从上下文中取出
	db := s.GetDB(ctx).Table(model.StudentCourseTableName).WithContext(ctx)
	err := db.Select("cla_id", "is_manually_added", "note").
		Where("stu_id = ? AND year = ? AND semester = ? AND cla_id IN (?)", stuID, year, semester, claIds).
		Find(&results).Error
	if err != nil {
		logh.Errorf("GetClassMetaData 数据库查询失败: %v, stuID: %s", err, stuID)
		return res
	}

	// 将切片结果转换为 map
	for _, row := range results {
		res[row.ClaID] = model.ClassMetaData{
			Note:            row.Note,
			IsManuallyAdded: row.IsManuallyAdded,
		}
	}

	return res
}

func (s StudentCourseDAO) GetClassNum(ctx context.Context, stuID, year, semester string, isManuallyAdded bool) (num int64, err error) {
	db := s.GetDB(ctx).Table(model.StudentCourseTableName)
	err = db.Where("stu_id = ? AND year = ? AND semester = ? AND is_manually_added = ?", stuID, year, semester, isManuallyAdded).Count(&num).Error
	if err != nil {
		return 0, err
	}
	return num, nil
}

func (s StudentCourseDAO) SaveStudentAndCourseToDB(ctx context.Context, sc *model.StudentCourse) error {
	logh := logger.From(ctx)
	if sc == nil {
		logh.Warn("insert student_course 0 data")
		return nil
	}
	db := s.GetDB(ctx).Table(model.StudentCourseTableName).WithContext(ctx)
	err := db.Debug().Clauses(clause.OnConflict{DoNothing: true}).Create(sc).Error
	if err != nil {
		logh.Errorf("Mysql:create %v in %s failed: %v", sc, model.StudentCourseTableName, err)
		return errcode.ErrClassUpdate
	}
	return nil
}

func (s StudentCourseDAO) SaveManyStudentAndCourseToDB(ctx context.Context, scs []*model.StudentCourse) error {
	logh := logger.GetLoggerFromCtx(ctx)
	if len(scs) == 0 {
		logh.Warn("insert student_course 0 data")
		return nil
	}

	db := s.GetDB(ctx).Table(model.StudentCourseTableName).WithContext(ctx)

	if err := db.Debug().Clauses(clause.OnConflict{DoNothing: true}).Create(scs).Error; err != nil {
		logh.Errorf("Mysql:create %v in %s failed: %v", scs, model.StudentCourseTableName, err)
		return errcode.ErrCourseSave
	}
	return nil
}

func (s StudentCourseDAO) DeleteStudentAndCourseByTimeFromDB(ctx context.Context, stuID, year, semester string) error {
	logh := logger.GetLoggerFromCtx(ctx)
	db := s.GetDB(ctx).Table(model.StudentCourseTableName).WithContext(ctx)
	// 注意:只删除非手动添加的课程，即官方课程
	err := db.Debug().Where("year = ? AND semester = ? AND stu_id = ? AND is_manually_added = false", year, semester, stuID).Delete(&model.StudentCourse{}).Error
	if err != nil {
		logh.Errorf("Mysql:delete student_course by time from db failed: %v", err)
		return errcode.ErrClassDelete
	}
	return nil
}
