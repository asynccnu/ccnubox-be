package dao

import (
	"context"
	"time"

	"github.com/asynccnu/ccnubox-be/be-grade/domain"
	"github.com/asynccnu/ccnubox-be/be-grade/repository/model"
	"gorm.io/gorm"
)

type RankDAO interface {
	GetRankByTerm(ctx context.Context, data *domain.GetRankByTermReq) (*model.Rank, error)
	RankExist(ctx context.Context, studentId string, t *Period) bool
	StoreRank(ctx context.Context, rank *model.Rank) error
	GetUpdateRank(ctx context.Context, size int, lastId int64) ([]model.Rank, error)
	UpdateViewAt(ctx context.Context, id int64) error
	DeleteRankByStudentId(ctx context.Context, year string) error
	DeleteRankByViewAt(ctx context.Context, time time.Time) error
}
type rankDAO struct {
	db *gorm.DB
}
type Period struct {
	XnmBegin int64
	XnmEnd   int64
	XqmBegin int64
	XqmEnd   int64
}

func NewRankDAO(db *gorm.DB) RankDAO {
	return &rankDAO{db: db}
}

func (d *rankDAO) GetRankByTerm(ctx context.Context, data *domain.GetRankByTermReq) (*model.Rank, error) {
	var ans model.Rank
	err := d.db.WithContext(ctx).
		Where("student_id = ?", data.StudentId).
		Where("xnm_begin = ?", data.XnmBegin).
		Where("xqm_begin = ?", data.XqmBegin).
		Where("xnm_end = ?", data.XnmEnd).
		Where("xqm_end = ?", data.XqmEnd).
		First(&ans).Error

	if err != nil {
		return nil, err
	}

	err = d.UpdateViewAt(ctx, ans.Id)
	if err != nil {
		return nil, err
	}

	return &ans, err
}

// 更新查询时间
func (d *rankDAO) UpdateViewAt(ctx context.Context, id int64) error {
	return d.db.WithContext(ctx).Model(&model.Rank{}).Where("id = ?", id).Update("view_at", time.Now()).Error
}

func (d *rankDAO) RankExist(ctx context.Context, studentId string, t *Period) bool {
	var count int64
	d.db.WithContext(ctx).Model(&model.Rank{}).
		Where("student_id = ?", studentId).
		Where("xqm_begin = ?", t.XqmBegin).
		Where("xqm_end = ?", t.XqmEnd).
		Where("xnm_begin = ?", t.XnmBegin).
		Where("xnm_end = ?", t.XnmEnd).
		Count(&count)

	return count > 0
}

func (d *rankDAO) StoreRank(ctx context.Context, rank *model.Rank) error {
	t := &Period{
		XqmBegin: rank.XqmBegin,
		XqmEnd:   rank.XqmEnd,
		XnmBegin: rank.XnmBegin,
		XnmEnd:   rank.XnmEnd,
	}

	if !d.RankExist(ctx, rank.StudentId, t) {
		return d.db.WithContext(ctx).Model(&model.Rank{}).Create(rank).Error
	}

	return d.db.WithContext(ctx).
		Model(&model.Rank{}).
		Where("student_id = ? AND xnm_begin = ? AND xqm_begin = ? AND xnm_end = ? AND xqm_end = ?",
			rank.StudentId, rank.XnmBegin, rank.XqmBegin, rank.XnmEnd, rank.XqmEnd).
		Updates(map[string]interface{}{
			"rank":    rank.Rank,
			"score":   rank.Score,
			"include": rank.Include,
			"update":  rank.Update,
		}).Error
}

func (d *rankDAO) GetUpdateRank(ctx context.Context, size int, lastId int64) ([]model.Rank, error) {
	var data []model.Rank
	// lastId保证不重复搜索数据，student_id排序原因见cron中的rank.go
	err := d.db.WithContext(ctx).Model(&model.Rank{}).
		Where("`update` = ?", true).
		Where("id > ?", lastId).
		Order("id ASC").
		Limit(size).Find(&data).Error

	return data, err
}

func (d *rankDAO) DeleteRankByStudentId(ctx context.Context, year string) error {
	return d.db.WithContext(ctx).Where("student_id <= ?", year).Delete(&model.Rank{}).Error
}

func (d *rankDAO) DeleteRankByViewAt(ctx context.Context, time time.Time) error {
	// 总rank不删
	return d.db.WithContext(ctx).Not("xnm_begin = ?", "2005").Where("view_at < ?", time).Delete(&model.Rank{}).Error
}
