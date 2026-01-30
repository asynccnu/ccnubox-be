package data

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/be-classlist/internal/data/do"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/errcode"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ClassInfoDBRepo struct {
	data *Data
}

func NewClassInfoDBRepo(data *Data) *ClassInfoDBRepo {
	return &ClassInfoDBRepo{
		data: data,
	}
}

func (c ClassInfoDBRepo) SaveClassInfosToDB(ctx context.Context, classInfos []*do.ClassInfo) error {
	logh := logger.GetLoggerFromCtx(ctx)
	if len(classInfos) == 0 {
		return nil
	}

	db := c.data.DB(ctx).Table(do.ClassInfoTableName).WithContext(ctx)
	err := db.Debug().
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"nature",
				"jxb_id",
			}),
		}).
		Create(&classInfos).Error
	if err != nil {
		logh.Errorf("Mysql:create %v in %s failed: %v", classInfos, do.ClassInfoTableName, err)
		return err
	}
	return nil
}

func (c ClassInfoDBRepo) AddClassInfoToDB(ctx context.Context, classInfo *do.ClassInfo) error {
	logh := logger.GetLoggerFromCtx(ctx)
	if classInfo == nil {
		return nil
	}
	if classInfo.Day < 1 || classInfo.Day > 7 {
		logh.Errorf("Mysql:create %v in %s failed: %v", classInfo, do.ClassInfoTableName, fmt.Errorf("date must between 1 and 7"))
		return errcode.ErrClassUpdate
	}
	db := c.data.DB(ctx).Table(do.ClassInfoTableName).WithContext(ctx)
	err := db.Debug().Clauses(clause.OnConflict{DoNothing: true}).Create(&classInfo).Error
	if err != nil {
		logh.Errorf("Mysql:create %v in %s failed: %v", classInfo, do.ClassInfoTableName, err)
		return errcode.ErrClassUpdate
	}
	return nil
}

func (c ClassInfoDBRepo) GetClassInfoFromDB(ctx context.Context, ID string) (*do.ClassInfo, error) {
	logh := logger.GetLoggerFromCtx(ctx)
	db := c.data.Mysql.Table(do.ClassInfoTableName).WithContext(ctx)
	cla := &do.ClassInfo{}
	err := db.Where("id =?", ID).First(cla).Error
	if err != nil {
		logh.Errorf("Mysql:find classinfo in %s where (id = %s) failed: %v", do.ClassInfoTableName, ID, err)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrClassNotFound
		}
		return nil, errcode.ErrClassFound
	}
	return cla, err
}

func (c ClassInfoDBRepo) GetClassInfos(ctx context.Context, stuId, xnm, xqm string) ([]*do.ClassInfo, error) {
	logh := logger.GetLoggerFromCtx(ctx)
	db := c.data.Mysql.WithContext(ctx)
	cla := make([]*do.ClassInfo, 0)

	err := db.Table(do.ClassInfoTableName).Select(fmt.Sprintf("%s.*,%s.note", do.ClassInfoTableName, do.StudentCourseTableName)).
		Joins(fmt.Sprintf(
			`LEFT JOIN %s ON %s.id = %s.cla_id`, do.StudentCourseTableName, do.ClassInfoTableName, do.StudentCourseTableName,
		)).
		Where(fmt.Sprintf(
			`%s.stu_id = ? AND %s.year = ? AND %s.semester = ?`, do.StudentCourseTableName, do.StudentCourseTableName, do.StudentCourseTableName),
			stuId, xnm, xqm,
		).
		Find(&cla).Error
	if err != nil {
		logh.Errorf("Mysql:find classinfos  where (stu_id = %s,year = %s,semester = %s) failed:%v",
			stuId, xnm, xqm, err)
		return nil, err
	}
	if len(cla) == 0 {
		logh.Warnf("Mysql:no classlist has been found,stuID:%s,year:%s,semester:%s failed: %v", stuId, xnm, xqm, err)
		return nil, nil
	}
	return cla, nil
}

func (c ClassInfoDBRepo) GetAllClassInfos(ctx context.Context, xnm, xqm string, cursor time.Time) ([]*do.ClassInfo, error) {
	logh := logger.GetLoggerFromCtx(ctx)
	db := c.data.Mysql.WithContext(ctx)
	cla := make([]*do.ClassInfo, 0)
	err := db.Table(do.ClassInfoTableName).
		Where("created_at > ? and id in (select distinct cla_id from student_course where year = ? and semester = ? and is_manually_added = ?)",
			cursor, xnm, xqm, false).
		Order("created_at ASC").
		Limit(100). // 最多100个
		Find(&cla).Error
	if err != nil {
		logh.Errorf("Mysql:find classinfos  where (is_manually_added = %v,year = %s,semester = %s) failed:%v",
			false, xnm, xqm, err)
		return nil, err
	}
	return cla, nil
}

func (c ClassInfoDBRepo) GetAddedClassInfos(ctx context.Context, stuID, xnm, xqm string) ([]*do.ClassInfo, error) {
	logh := logger.GetLoggerFromCtx(ctx)
	db := c.data.Mysql.WithContext(ctx)
	cla := make([]*do.ClassInfo, 0)
	err := db.Table(do.ClassInfoTableName).Select(fmt.Sprintf("%s.*", do.ClassInfoTableName)).
		Joins(fmt.Sprintf(
			`LEFT JOIN %s ON %s.id = %s.cla_id`, do.StudentCourseTableName, do.ClassInfoTableName, do.StudentCourseTableName,
		)).
		Where(fmt.Sprintf(
			`%s.stu_id = ? AND %s.year = ? AND %s.semester = ? AND %s.is_manually_added =?`, do.StudentCourseTableName, do.StudentCourseTableName, do.StudentCourseTableName, do.StudentCourseTableName),
			stuID, xnm, xqm, true,
		).Find(&cla).Error
	if err != nil {
		logh.Errorf("mysql failed to find added class_infos[%v,%v,%v]: %v", stuID, xnm, xqm, err)
		return nil, err
	}
	return cla, nil
}

func (c ClassInfoDBRepo) GetClassNaturesFromDB(ctx context.Context, stuID string) ([]string, error) {
	logh := logger.GetLoggerFromCtx(ctx)
	db := c.data.Mysql.WithContext(ctx)

	var natures []string

	subQuery := db.
		Table(do.StudentCourseTableName).
		Select("cla_id").
		Where("stu_id = ?", stuID)

	err := db.
		Table(do.ClassInfoTableName).
		Distinct("nature").
		Where("id IN (?) AND nature IS NOT NULL", subQuery).
		Pluck("nature", &natures).Error
	if err != nil {
		logh.Errorf("mysql failed to find classNatures from db: %v", err)
		return nil, err
	}
	return natures, nil
}
