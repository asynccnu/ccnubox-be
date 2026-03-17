package dao

import (
	"context"
	"fmt"

	"github.com/asynccnu/ccnubox-be/be-classlist_v2/biz/errcode"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/repository/model"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"gorm.io/gorm/clause"
)

type ClassInfoDAO struct {
	BaseDAO
}

func NewClassInfoDAO(base BaseDAO) *ClassInfoDAO {
	return &ClassInfoDAO{BaseDAO: base}
}

func (c ClassInfoDAO) SaveClassInfosToDB(ctx context.Context, classInfos []*model.ClassInfo) error {
	logh := logger.From(ctx)
	if len(classInfos) == 0 {
		logh.Warnf("no classinfo to save!")
		return nil
	}

	db := c.GetDB(ctx).Table(model.ClassInfoTableName).WithContext(ctx)
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
		return fmt.Errorf("Mysql:create %v in %s failed: %v", classInfos, model.ClassInfoTableName, err)
	}
	return nil
}

func (c ClassInfoDAO) AddClassInfoToDB(ctx context.Context, classInfo *model.ClassInfo) error {
	logh := logger.GetLoggerFromCtx(ctx)
	if classInfo == nil {
		return nil
	}
	if classInfo.Day < 1 || classInfo.Day > 7 {
		logh.Errorf("Mysql:create %v in %s failed: %v", classInfo, model.ClassInfoTableName, fmt.Errorf("date must between 1 and 7"))
		return errcode.ErrClassUpdate
	}

	// 约定将事务 db 从 ctx 中取出来
	db := c.GetDB(ctx).Table(model.ClassInfoTableName).WithContext(ctx)
	err := db.Debug().Clauses(clause.OnConflict{DoNothing: true}).Create(&classInfo).Error
	if err != nil {
		logh.Errorf("Mysql:create %v in %s failed: %v", classInfo, model.ClassInfoTableName, err)
		return errcode.ErrClassUpdate
	}
	return nil
}

func (c ClassInfoDAO) GetClassInfos(ctx context.Context, stuId, xnm, xqm string) ([]*model.ClassInfo, error) {
	logh := logger.From(ctx)
	db := c.GetDB(ctx).WithContext(ctx)
	cla := make([]*model.ClassInfo, 0)

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
		logh.Errorf("Mysql:find classinfos where (stu_id = %s,year = %s,semester = %s) failed:%v",
			stuId, xnm, xqm, err)
		return nil, err
	}
	if len(cla) == 0 {
		logh.Warnf("Mysql:no classlist has been found,stuID:%s,year:%s,semester:%s failed: %v", stuId, xnm, xqm, err)
		return nil, nil
	}
	return cla, nil
}

func (c ClassInfoDAO) GetAddedClassInfos(ctx context.Context, stuID, xnm, xqm string) ([]*model.ClassInfo, error) {
	logh := logger.GetLoggerFromCtx(ctx)
	db := c.GetDB(ctx)
	cla := make([]*model.ClassInfo, 0)
	err := db.Table(model.ClassInfoTableName).Select(fmt.Sprintf("%s.*", model.ClassInfoTableName)).
		Joins(fmt.Sprintf(
			`LEFT JOIN %s ON %s.id = %s.cla_id`, model.StudentCourseTableName, model.ClassInfoTableName, model.StudentCourseTableName,
		)).
		Where(fmt.Sprintf(
			`%s.stu_id = ? AND %s.year = ? AND %s.semester = ? AND %s.is_manually_added =?`, model.StudentCourseTableName, model.StudentCourseTableName, model.StudentCourseTableName, model.StudentCourseTableName),
			stuID, xnm, xqm, true,
		).Find(&cla).Error
	if err != nil {
		logh.Errorf("mysql failed to find added class_infos[%v,%v,%v]: %v", stuID, xnm, xqm, err)
		return nil, err
	}
	return cla, nil
}
