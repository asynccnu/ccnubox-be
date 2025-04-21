package data

import (
	"context"
	"errors"
	"fmt"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/classLog"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/errcode"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

type ClassInfoDBRepo struct {
	data *Data
	log  classLog.Clogger
}

func NewClassInfoDBRepo(data *Data, logger classLog.Clogger) *ClassInfoDBRepo {
	return &ClassInfoDBRepo{
		log:  logger,
		data: data,
	}
}
func (c ClassInfoDBRepo) SaveClassInfosToDB(ctx context.Context, classInfos []*model.ClassInfo) error {
	if len(classInfos) == 0 {
		return nil
	}

	db := c.data.DB(ctx).Table(model.ClassInfoTableName).WithContext(ctx)
	err := db.Debug().Clauses(clause.OnConflict{DoNothing: true}).Create(&classInfos).Error
	if err != nil {
		c.log.Errorw(classLog.Msg, fmt.Sprintf("Mysql:create %v in %s", classInfos, model.ClassInfoTableName),
			classLog.Reason, err)
		return err
	}
	return nil
}

func (c ClassInfoDBRepo) AddClassInfoToDB(ctx context.Context, classInfo *model.ClassInfo) error {
	if classInfo == nil {
		return nil
	}
	db := c.data.DB(ctx).Table(model.ClassInfoTableName).WithContext(ctx)
	err := db.Debug().Clauses(clause.OnConflict{DoNothing: true}).Create(&classInfo).Error
	if err != nil {
		c.log.Errorw(classLog.Msg, fmt.Sprintf("Mysql:create %v in %s", classInfo, model.ClassInfoTableName),
			classLog.Reason, err)
		return errcode.ErrClassUpdate
	}
	return nil
}

func (c ClassInfoDBRepo) GetClassInfoFromDB(ctx context.Context, ID string) (*model.ClassInfo, error) {
	db := c.data.Mysql.Table(model.ClassInfoTableName).WithContext(ctx)
	cla := &model.ClassInfo{}
	err := db.Where("id =?", ID).First(cla).Error
	if err != nil {
		c.log.Errorw(classLog.Msg, fmt.Sprintf("Mysql:find classinfo in %s where (id = %s)", model.ClassInfoTableName, ID),
			classLog.Reason, err)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrClassNotFound
		}
		return nil, errcode.ErrClassFound
	}
	return cla, err
}

func (c ClassInfoDBRepo) GetClassInfos(ctx context.Context, stuId, xnm, xqm string) ([]*model.ClassInfo, error) {
	db := c.data.Mysql.WithContext(ctx)
	var (
		cla = make([]*model.ClassInfo, 0)
	)
	err := db.Table(model.ClassInfoTableName).Select(fmt.Sprintf("%s.*", model.ClassInfoTableName)).
		Joins(fmt.Sprintf(
			`LEFT JOIN %s ON %s.id = %s.cla_id`, model.StudentCourseTableName, model.ClassInfoTableName, model.StudentCourseTableName,
		)).
		Where(fmt.Sprintf(
			`%s.stu_id = ? AND %s.year = ? AND %s.semester = ?`, model.StudentCourseTableName, model.StudentCourseTableName, model.StudentCourseTableName),
			stuId, xnm, xqm,
		).
		Find(&cla).Error
	if err != nil {
		c.log.Errorw(classLog.Msg, fmt.Sprintf("Mysql:find classinfos  where (stu_id = %s,year = %s,semester = %s)",
			stuId, xnm, xqm),
			classLog.Reason, err)
		return nil, err
	}
	if len(cla) == 0 {
		c.log.Warnw(classLog.Msg, fmt.Sprintf("Mysql:no classlist has been found,stuID:%s,year:%s,semester:%s", stuId, xnm, xqm))
		return nil, nil
	}
	return cla, nil
}

func (c ClassInfoDBRepo) GetAllClassInfos(ctx context.Context, xnm, xqm string, cursor time.Time) ([]*model.ClassInfo, error) {
	db := c.data.Mysql.WithContext(ctx)
	var (
		cla = make([]*model.ClassInfo, 0)
	)
	err := db.Table(model.ClassInfoTableName).
		Where(fmt.Sprintf(
			`%s.year = ? AND %s.semester = ? AND %s.created_at > ?`, model.ClassInfoTableName, model.ClassInfoTableName, model.ClassInfoTableName),
			xnm, xqm, cursor,
		).
		Order(fmt.Sprintf(
			"%s.created_at ASC", model.ClassInfoTableName,
		)).
		Limit(100). //最多100个
		Find(&cla).Error

	if err != nil {
		c.log.Errorw(classLog.Msg, fmt.Sprintf("Mysql:find classinfos  where (is_manually_added = %v,year = %s,semester = %s)",
			false, xnm, xqm),
			classLog.Reason, err)
		return nil, err
	}
	return cla, nil
}

func (c ClassInfoDBRepo) GetAddedClassInfos(ctx context.Context, stuID, xnm, xqm string) ([]*model.ClassInfo, error) {
	db := c.data.Mysql.WithContext(ctx)
	var (
		cla = make([]*model.ClassInfo, 0)
	)
	err := db.Table(model.ClassInfoTableName).Select(fmt.Sprintf("%s.*", model.ClassInfoTableName)).
		Joins(fmt.Sprintf(
			`LEFT JOIN %s ON %s.id = %s.cla_id`, model.StudentCourseTableName, model.ClassInfoTableName, model.StudentCourseTableName,
		)).
		Where(fmt.Sprintf(
			`%s.stu_id = ? AND %s.year = ? AND %s.semester = ? AND %s.is_manually_added =?`, model.StudentCourseTableName, model.StudentCourseTableName, model.StudentCourseTableName, model.StudentCourseTableName),
			stuID, xnm, xqm, true,
		).Find(&cla).Error
	if err != nil {
		c.log.Errorw(classLog.Msg, fmt.Sprintf("mysql failed to find added class_infos[%v,%v,%v]: %v", stuID, xnm, xqm, err))
		return nil, err
	}
	return cla, nil
}
