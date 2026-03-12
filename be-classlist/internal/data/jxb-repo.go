package data

import (
	"context"

	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"gorm.io/gorm/clause"
)

type JxbDBRepo struct {
	data *Data
}

func NewJxbDBRepo(data *Data) *JxbDBRepo {
	return &JxbDBRepo{
		data: data,
	}
}

func (j *JxbDBRepo) SaveJxb(ctx context.Context, stuID string, jxbID []string) error {
	logh := logger.GetLoggerFromCtx(ctx).WithContext(ctx)

	if len(jxbID) == 0 {
		return nil
	}

	db := j.data.Mysql.Table(JxbTableName).WithContext(ctx)
	jxb := make([]Jxb, 0, len(jxbID))
	for _, id := range jxbID {
		jxb = append(jxb, Jxb{
			JxbId: id,
			StuId: stuID,
		})
	}

	err := db.Debug().Clauses(clause.OnConflict{DoNothing: true}).Create(&jxb).Error
	if err != nil {
		logh.Errorf("Mysql:create %v in %s failed: %v", jxb, JxbTableName, err)
		return err
	}
	return nil
}

func (j *JxbDBRepo) FindStuIdsByJxbId(ctx context.Context, jxbId string) ([]string, error) {
	logh := logger.GetLoggerFromCtx(ctx).WithContext(ctx)

	var stuIds []string
	err := j.data.Mysql.Table(JxbTableName).WithContext(ctx).
		Select("stu_id").Where("jxb_id = ?", jxbId).Find(&stuIds).Error
	if err != nil {
		logh.Errorf("Mysql:find stu_id in %s where (jxb_id = %s) failed: %v", JxbTableName, jxbId, err)
		return nil, err
	}
	return stuIds, nil
}
