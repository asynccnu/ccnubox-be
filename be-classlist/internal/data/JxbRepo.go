package data

import (
	"context"
	"fmt"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/classLog"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/model"
	"gorm.io/gorm/clause"
)

type JxbDBRepo struct {
	data *Data
	log  classLog.Clogger
}

func NewJxbDBRepo(data *Data, logger classLog.Clogger) *JxbDBRepo {
	return &JxbDBRepo{
		data: data,
		log:  logger,
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
		j.log.Errorw(classLog.Msg, fmt.Sprintf("Mysql:Save Jxb{%+v} err)", jxb),
			classLog.Reason, err)
		return err
	}
	return nil
}
func (j *JxbDBRepo) FindStuIdsByJxbId(ctx context.Context, jxbId string) ([]string, error) {
	var stuIds []string
	err := j.data.Mysql.Raw("SELECT stu_id FROM jxb WHERE jxb_id =  ?", jxbId).Scan(&stuIds).Error
	if err != nil {
		j.log.Errorw(classLog.Msg, fmt.Sprintf("Mysql:Find StuIds By JxbId(%s) err", jxbId),
			classLog.Reason, err)
		return nil, err
	}
	return stuIds, nil
}
