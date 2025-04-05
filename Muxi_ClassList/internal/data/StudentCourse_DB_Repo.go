package data

import (
	"context"
	"errors"
	"fmt"
	"github.com/asynccnu/ccnubox-be/Muxi_ClassList/internal/classLog"
	"github.com/asynccnu/ccnubox-be/Muxi_ClassList/internal/errcode"
	"github.com/asynccnu/ccnubox-be/Muxi_ClassList/internal/model"
	"gorm.io/gorm/clause"
)

type StudentAndCourseDBRepo struct {
	data *Data
	log  classLog.Clogger
}

func NewStudentAndCourseDBRepo(data *Data, logger classLog.Clogger) *StudentAndCourseDBRepo {
	return &StudentAndCourseDBRepo{
		log:  logger,
		data: data,
	}
}

func (s StudentAndCourseDBRepo) SaveManyStudentAndCourseToDB(ctx context.Context, scs []*model.StudentCourse) error {

	if len(scs) == 0 {
		return nil
	}

	db := s.data.DB(ctx).Table(model.StudentCourseTableName).WithContext(ctx)

	// 处理 StudentCourse
	for _, sc := range scs {
		if err := db.Debug().Clauses(clause.OnConflict{DoNothing: true}).Create(sc).Error; err != nil {
			s.log.Errorw(classLog.Msg, fmt.Sprintf("Mysql:create StudentAndCourses(%v)", sc),
				classLog.Reason, err)
			return errcode.ErrCourseSave
		}
	}
	return nil
}

func (s StudentAndCourseDBRepo) SaveStudentAndCourseToDB(ctx context.Context, sc *model.StudentCourse) error {
	if sc == nil {
		return nil
	}

	db := s.data.DB(ctx).Table(model.StudentCourseTableName).WithContext(ctx)
	err := db.Debug().Clauses(clause.OnConflict{DoNothing: true}).Create(sc).Error
	if err != nil {
		s.log.Errorw(classLog.Msg, fmt.Sprintf("Mysql:create StudentAndCourse(%v)", sc),
			classLog.Reason, err)
		return errcode.ErrClassUpdate
	}
	return nil
}

func (s StudentAndCourseDBRepo) DeleteStudentAndCourseInDB(ctx context.Context, stuID, year, semester string, claID []string) error {
	if len(claID) == 0 {
		return errors.New("mysql can't delete zero data")
	}
	db := s.data.DB(ctx).Table(model.StudentCourseTableName).WithContext(ctx)
	err := db.Debug().Where("year = ? AND semester = ? AND stu_id = ? AND cla_id IN ?", year, semester, stuID, claID).Delete(&model.StudentCourse{}).Error
	if err != nil {
		return errcode.ErrClassDelete
	}
	return nil
}
func (s StudentAndCourseDBRepo) CheckExists(ctx context.Context, xnm, xqm, stuId, classId string) bool {
	db := s.data.Mysql.Table(model.StudentCourseTableName).WithContext(ctx)
	var cnt int64
	err := db.Where("stu_id = ? AND cla_id = ? AND year = ? AND semester = ?", stuId, classId, xnm, xqm).Count(&cnt).Error
	if err != nil || cnt == 0 {
		return false
	}
	return true
}

func (s StudentAndCourseDBRepo) GetClassNum(ctx context.Context, stuID, year, semester string, isManuallyAdded bool) (num int64, err error) {
	db := s.data.DB(ctx).Table(model.StudentCourseTableName)
	err = db.Where("stu_id = ? AND year = ? AND semester = ? AND is_manually_added = ?", stuID, year, semester, isManuallyAdded).Count(&num).Error
	if err != nil {
		return 0, err
	}
	return num, nil
}

func (s StudentAndCourseDBRepo) DeleteStudentAndCourseByTimeFromDB(ctx context.Context, stuID, year, semester string) error {
	db := s.data.DB(ctx).Table(model.StudentCourseTableName).WithContext(ctx)
	//注意:只删除非手动添加的课程，即官方课程
	err := db.Debug().Where("year = ? AND semester = ? AND stu_id = ? AND is_manually_added = false", year, semester, stuID).Delete(&model.StudentCourse{}).Error
	if err != nil {
		return errcode.ErrClassDelete
	}
	return nil
}
