package dao

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-classlist_v2/repository/model"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
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

	if err := db.Debug().Clauses(clause.OnConflict{DoNothing: true}).Create(&jxb).Error; err != nil {
		return errorx.Errorf("dao.jxb.SaveJxb: stuID=%s, jxbIDs=%v, table=%s: %w", stuID, jxbID, model.JxbTableName, err)
	}
	return nil
}

func (j *JxbDAO) FindStuIdsByJxbId(ctx context.Context, jxbId string) ([]string, error) {
	var stuIds []string
	err := j.GetDB(ctx).Table(model.JxbTableName).WithContext(ctx).
		Select("stu_id").Where("jxb_id = ?", jxbId).Find(&stuIds).Error
	if err != nil {
		return nil, errorx.Errorf("dao.jxb.FindStuIdsByJxbId: jxbId=%s, table=%s: %w", jxbId, model.JxbTableName, err)
	}
	return stuIds, nil
}
