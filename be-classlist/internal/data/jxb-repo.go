package data

import (
	"context"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/model"
	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm/clause"
)

type JxbDBRepo struct {
	data *Data
	log  *log.Helper
}

func NewJxbDBRepo(data *Data, logger log.Logger) *JxbDBRepo {
	return &JxbDBRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (j *JxbDBRepo) SaveJxb(ctx context.Context, stuID string, jxbID []string) error {
	if len(jxbID) == 0 {
		return nil
	}

	db := j.data.Mysql.Table(model.JxbTableName).WithContext(ctx)
	var jxb = make([]model.Jxb, 0, len(jxbID))
	for _, id := range jxbID {
		jxb = append(jxb, model.Jxb{
			JxbId: id,
			StuId: stuID,
		})
	}

	err := db.Debug().Clauses(clause.OnConflict{DoNothing: true}).Create(&jxb).Error
	if err != nil {
		j.log.Errorf("Mysql:create %v in %s failed: %v", jxb, model.JxbTableName, err)
		return err
	}
	return nil
}
func (j *JxbDBRepo) FindStuIdsByJxbId(ctx context.Context, jxbId string) ([]string, error) {
	var stuIds []string
	err := j.data.Mysql.Table(model.JxbTableName).WithContext(ctx).
		Select("stu_id").Where("jxb_id = ?", jxbId).Find(&stuIds).Error
	if err != nil {
		j.log.Errorf("Mysql:find stu_id in %s where (jxb_id = %s) failed: %v", model.JxbTableName, jxbId, err)
		return nil, err
	}
	return stuIds, nil
}
