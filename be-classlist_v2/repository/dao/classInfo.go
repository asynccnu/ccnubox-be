package dao

import (
	"context"
	"fmt"

	"github.com/asynccnu/ccnubox-be/be-classlist_v2/biz/errcode"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/repository/model"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"gorm.io/gorm/clause"
)

type ClassInfoDAO struct {
	BaseDAO
	log logger.Logger
}

func NewClassInfoDAO(base BaseDAO, l logger.Logger) *ClassInfoDAO {
	return &ClassInfoDAO{BaseDAO: base, log: l}
}

func (c ClassInfoDAO) SaveClassInfosToDB(ctx context.Context, classInfos []*model.ClassInfo) error {
	if len(classInfos) == 0 {
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
		return errorx.Errorf("dao.classInfo.SaveClassInfosToDB: count=%d, table=%s: %w", len(classInfos), model.ClassInfoTableName, err)
	}
	return nil
}

func (c ClassInfoDAO) AddClassInfoToDB(ctx context.Context, classInfo *model.ClassInfo) error {
	if classInfo == nil {
		return nil
	}
	if classInfo.Day < 1 || classInfo.Day > 7 {
		return errorx.Errorf("dao.classInfo.AddClassInfoToDB: invalid day=%d, classInfo=%+v: %w", classInfo.Day, classInfo, errcode.ErrClassUpdate)
	}

	// 约定将事务 db 从 ctx 中取出来
	db := c.GetDB(ctx).Table(model.ClassInfoTableName).WithContext(ctx)
	err := db.Debug().Clauses(clause.OnConflict{DoNothing: true}).Create(&classInfo).Error
	if err != nil {
		return errorx.Errorf("dao.classInfo.AddClassInfoToDB: classInfo=%+v, dbErr=%s: %w", classInfo, err.Error(), errcode.ErrClassUpdate)
	}
	return nil
}

func (c ClassInfoDAO) DeleteAddedClassInfos(ctx context.Context, classIDs []string) error {
	if len(classIDs) == 0 {
		return nil
	}

	db := c.GetDB(ctx).Table(model.ClassInfoTableName).WithContext(ctx)
	err := db.Debug().
		Where("id IN ?", classIDs).
		Delete(&model.ClassInfo{}).Error
	if err != nil {
		return errorx.Errorf("dao.classInfo.DeleteAddedClassInfos: classIDs=%v: %w", classIDs, errcode.ErrClassDelete)
	}
	return nil
}

func (c ClassInfoDAO) GetClassInfos(ctx context.Context, stuId, xnm, xqm string) ([]*model.ClassInfo, error) {
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
		return nil, errorx.Errorf("dao.classInfo.GetClassInfos: stuID=%s, year=%s, semester=%s: %w", stuId, xnm, xqm, err)
	}
	if len(cla) == 0 {
		return nil, nil
	}
	return cla, nil
}

func (c ClassInfoDAO) GetAddedClassInfos(ctx context.Context, stuID, xnm, xqm string) ([]*model.ClassInfo, error) {
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
		return nil, errorx.Errorf("dao.classInfo.GetAddedClassInfos: stuID=%s, year=%s, semester=%s: %w", stuID, xnm, xqm, err)
	}
	return cla, nil
}
