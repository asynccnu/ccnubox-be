package dao

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-classlist_v2/repository/model"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"gorm.io/gorm/clause"
)

type JxbDAO struct {
	BaseDAO
	log logger.Logger
}

func NewJxbDAO(base BaseDAO, l logger.Logger) *JxbDAO {
	return &JxbDAO{
		BaseDAO: base,
		log:     l,
	}
}

func (j *JxbDAO) SaveJxb(ctx context.Context, stuID string, jxbID []string) error {
	logh := j.log.WithContext(ctx)

	if len(jxbID) == 0 {
		return nil
	}

	db := j.GetDB(ctx).Table(model.JxbTableName).WithContext(ctx)
	jxb := make([]model.Jxb, 0, len(jxbID))
	for _, id := range jxbID {
		jxb = append(jxb, model.Jxb{
			JxbId: id,
			StuId: stuID,
		})
	}

	err := db.Debug().Clauses(clause.OnConflict{DoNothing: true}).Create(&jxb).Error
	if err != nil {
		logh.Errorf("Mysql:create %v in %s failed: %v", jxb, model.JxbTableName, err)
		return err
	}
	return nil
}

func (j *JxbDAO) FindStuIdsByJxbId(ctx context.Context, jxbId string) ([]string, error) {
	logh := j.log.WithContext(ctx)

	var stuIds []string
	err := j.GetDB(ctx).Table(model.JxbTableName).WithContext(ctx).
		Select("stu_id").Where("jxb_id = ?", jxbId).Find(&stuIds).Error
	if err != nil {
		logh.Errorf("Mysql:find stu_id in %s where (jxb_id = %s) failed: %v", model.JxbTableName, jxbId, err)
		return nil, err
	}
	return stuIds, nil
}
