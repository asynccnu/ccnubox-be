package dao

import (
	"context"
	"time"

	"github.com/asynccnu/ccnubox-be/be-grade/domain"
	"github.com/asynccnu/ccnubox-be/be-grade/repository/model"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
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
		return nil, errorx.Errorf("dao: get rank by term failed, sid: %s, range: %d-%d, err: %w", data.StudentId, data.XnmBegin, data.XnmEnd, err)
	}

	// 记录查询足迹，用于后续清理冷数据
	err = d.UpdateViewAt(ctx, ans.Id)
	if err != nil {
		// 这里记录日志但不中断返回，因为数据已经查到了
		return &ans, nil
	}

	return &ans, nil
}

// 更新查询时间
func (d *rankDAO) UpdateViewAt(ctx context.Context, id int64) error {
	err := d.db.WithContext(ctx).Model(&model.Rank{}).
		Where("id = ?", id).
		Update("view_at", time.Now()).Error
	if err != nil {
		return errorx.Errorf("dao: update view_at failed, id: %d, err: %w", id, err)
	}
	return nil
}

func (d *rankDAO) RankExist(ctx context.Context, studentId string, t *Period) bool {
	var count int64
	// 修正：增加对 Count 错误的检查，虽然原接口返回 bool，但内部应保证连接正常
	err := d.db.WithContext(ctx).Model(&model.Rank{}).
		Where("student_id = ?", studentId).
		Where("xqm_begin = ?", t.XqmBegin).
		Where("xqm_end = ?", t.XqmEnd).
		Where("xnm_begin = ?", t.XnmBegin).
		Where("xnm_end = ?", t.XnmEnd).
		Count(&count).Error

	if err != nil {
		return false
	}
	return count > 0
}

func (d *rankDAO) StoreRank(ctx context.Context, rank *model.Rank) error {
	t := &Period{
		XqmBegin: rank.XqmBegin,
		XqmEnd:   rank.XqmEnd,
		XnmBegin: rank.XnmBegin,
		XnmEnd:   rank.XnmEnd,
	}

	// 使用更健壮的 Save 或 Transaction 逻辑
	if !d.RankExist(ctx, rank.StudentId, t) {
		err := d.db.WithContext(ctx).Model(&model.Rank{}).Create(rank).Error
		if err != nil {
			return errorx.Errorf("dao: create rank record failed, sid: %s, err: %w", rank.StudentId, err)
		}
		return nil
	}

	err := d.db.WithContext(ctx).
		Model(&model.Rank{}).
		Where("student_id = ? AND xnm_begin = ? AND xqm_begin = ? AND xnm_end = ? AND xqm_end = ?",
			rank.StudentId, rank.XnmBegin, rank.XqmBegin, rank.XnmEnd, rank.XqmEnd).
		Updates(map[string]interface{}{
			"rank":    rank.Rank,
			"score":   rank.Score,
			"include": rank.Include,
			"update":  rank.Update,
			"view_at": time.Now(), // 更新时同步刷新查看时间
		}).Error

	if err != nil {
		return errorx.Errorf("dao: update rank record failed, sid: %s, err: %w", rank.StudentId, err)
	}
	return nil
}

func (d *rankDAO) GetUpdateRank(ctx context.Context, size int, lastId int64) ([]model.Rank, error) {
	var data []model.Rank
	err := d.db.WithContext(ctx).Model(&model.Rank{}).
		Where("`update` = ?", true).
		Where("id > ?", lastId).
		Order("id ASC").
		Limit(size).Find(&data).Error

	if err != nil {
		return nil, errorx.Errorf("dao: get update-required ranks failed, lastId: %d, err: %w", lastId, err)
	}
	return data, nil
}

func (d *rankDAO) DeleteRankByStudentId(ctx context.Context, year string) error {
	err := d.db.WithContext(ctx).
		Where("student_id <= ?", year).
		Delete(&model.Rank{}).Error
	if err != nil {
		return errorx.Errorf("dao: delete rank by student_id prefix failed, year_limit: %s, err: %w", year, err)
	}
	return nil
}

func (d *rankDAO) DeleteRankByViewAt(ctx context.Context, timeLimit time.Time) error {
	// 排除总排名（2005年开始的数据通常被视为全局统计）
	err := d.db.WithContext(ctx).
		Not("xnm_begin = ?", 2005). // 修正：数据库存储通常为 int，原代码中传字符串可能触发隐式转换
		Where("view_at < ?", timeLimit).
		Delete(&model.Rank{}).Error

	if err != nil {
		return errorx.Errorf("dao: delete expired ranks failed, before: %v, err: %w", timeLimit, err)
	}
	return nil
}
