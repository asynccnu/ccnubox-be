package data

import (
	"context"
	"errors"

	"github.com/asynccnu/ccnubox-be/be-classlist/internal/data/do"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/errcode"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"gorm.io/gorm/clause"
)

type StudentAndCourseDBRepo struct {
	data *Data
}

func NewStudentAndCourseDBRepo(data *Data) *StudentAndCourseDBRepo {
	return &StudentAndCourseDBRepo{
		data: data,
	}
}

func (s StudentAndCourseDBRepo) SaveManyStudentAndCourseToDB(ctx context.Context, scs []*do.StudentCourse) error {
	logh := logger.GetLoggerFromCtx(ctx)
	if len(scs) == 0 {
		logh.Warn("insert student_course 0 data")
		return nil
	}

	db := s.data.DB(ctx).Table(do.StudentCourseTableName).WithContext(ctx)

	if err := db.Debug().Clauses(clause.OnConflict{DoNothing: true}).Create(scs).Error; err != nil {
		logh.Errorf("Mysql:create %v in %s failed: %v", scs, do.StudentCourseTableName, err)
		return errcode.ErrCourseSave
	}
	return nil
}

func (s StudentAndCourseDBRepo) SaveStudentAndCourseToDB(ctx context.Context, sc *do.StudentCourse) error {
	logh := logger.GetLoggerFromCtx(ctx)
	if sc == nil {
		logh.Warn("insert student_course 0 data")
		return nil
	}
	db := s.data.DB(ctx).Table(do.StudentCourseTableName).WithContext(ctx)
	err := db.Debug().Clauses(clause.OnConflict{DoNothing: true}).Create(sc).Error
	if err != nil {
		logh.Errorf("Mysql:create %v in %s failed: %v", sc, do.StudentCourseTableName, err)
		return errcode.ErrClassUpdate
	}
	return nil
}

func (s StudentAndCourseDBRepo) DeleteStudentAndCourseInDB(ctx context.Context, stuID, year, semester string, claID []string) error {
	logh := logger.GetLoggerFromCtx(ctx)
	if len(claID) == 0 {
		logh.Warn("delete student_course 0 data")
		return errors.New("mysql can't delete zero data")
	}
	db := s.data.DB(ctx).Table(do.StudentCourseTableName).WithContext(ctx)
	err := db.Debug().Where("year = ? AND semester = ? AND stu_id = ? AND cla_id IN (?)", year, semester, stuID, claID).Delete(&do.StudentCourse{}).Error
	if err != nil {
		logh.Errorf("Mysql:delete %v in %s failed: %v", claID, do.StudentCourseTableName, err)
		return errcode.ErrClassDelete
	}
	return nil
}

func (s StudentAndCourseDBRepo) CheckExists(ctx context.Context, xnm, xqm, stuId, classId string) bool {
	db := s.data.Mysql.Table(do.StudentCourseTableName).WithContext(ctx)
	var cnt int64
	err := db.Where("stu_id = ?  AND year = ? AND semester = ? AND cla_id = ?", stuId, xnm, xqm, classId).Count(&cnt).Error
	if err != nil || cnt == 0 {
		return false
	}
	return true
}

func (s StudentAndCourseDBRepo) GetClassNum(ctx context.Context, stuID, year, semester string, isManuallyAdded bool) (num int64, err error) {
	db := s.data.DB(ctx).Table(do.StudentCourseTableName)
	err = db.Where("stu_id = ? AND year = ? AND semester = ? AND is_manually_added = ?", stuID, year, semester, isManuallyAdded).Count(&num).Error
	if err != nil {
		return 0, err
	}
	return num, nil
}

func (s StudentAndCourseDBRepo) DeleteStudentAndCourseByTimeFromDB(ctx context.Context, stuID, year, semester string) error {
	logh := logger.GetLoggerFromCtx(ctx)
	db := s.data.DB(ctx).Table(do.StudentCourseTableName).WithContext(ctx)
	// 注意:只删除非手动添加的课程，即官方课程
	err := db.Debug().Where("year = ? AND semester = ? AND stu_id = ? AND is_manually_added = false", year, semester, stuID).Delete(&do.StudentCourse{}).Error
	if err != nil {
		logh.Errorf("Mysql:delete student_course by time from db failed: %v", err)
		return errcode.ErrClassDelete
	}
	return nil
}

func (s StudentAndCourseDBRepo) CheckManualCourseStatus(ctx context.Context, stuID, year, semester, classID string) bool {
	db := s.data.DB(ctx).Table(do.StudentCourseTableName).WithContext(ctx)

	var isAdded bool

	err := db.Where("stu_id = ? and year = ? and semester = ? and cla_id = ?", stuID, year, semester, classID).
		Pluck("is_manually_added", &isAdded).Error
	if err != nil {
		return true
	}
	return isAdded
}

func (s StudentAndCourseDBRepo) GetCourseNote(ctx context.Context, stuID, year, semester, classID string) string {
	logh := logger.GetLoggerFromCtx(ctx)
	db := s.data.DB(ctx).Table(do.StudentCourseTableName).WithContext(ctx)

	var note string

	err := db.
		Select("note").
		Where("stu_id = ? and year = ? and semester = ? and cla_id = ?", stuID, year, semester, classID).
		Limit(1).
		Scan(&note).Error
	if err != nil {
		logh.Warnf("Query course_note failed[stu_id:%v year:%v semester:%v cla_id:%v]: %v", stuID, year, semester, classID, err)
		return ""
	}

	if note == "" {
		logh.Warnf("Course note empty[stu_id:%v year:%v semester:%v cla_id:%v]", stuID, year, semester, classID)
	} else {
		logh.Infof("Course note found[stu_id:%v year:%v semester:%v cla_id:%v] len=%d", stuID, year, semester, classID, len(note))
	}

	return note
}

func (s StudentAndCourseDBRepo) UpdateCourseNote(ctx context.Context, stuID, year, semester, classID, note string) error {
	db := s.data.DB(ctx).Table(do.StudentCourseTableName).WithContext(ctx)

	cn := &do.StudentCourse{StuID: stuID, Year: year, Semester: semester, ClaID: classID, Note: note}

	err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "stu_id"}, {Name: "year"}, {Name: "semester"}, {Name: "cla_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{"note": note}),
	}).Create(cn).Error
	if err != nil {
		return errcode.ErrClassUpdate
	}
	return nil
}
