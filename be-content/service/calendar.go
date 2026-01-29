package service

import (
	"context"
	"errors"
	"github.com/asynccnu/ccnubox-be/be-content/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/be-content/repository"

	"github.com/asynccnu/ccnubox-be/be-content/domain"
	"github.com/asynccnu/ccnubox-be/be-content/repository/model"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

// 定义接口
type CalendarService interface {
	GetList(ctx context.Context) ([]domain.Calendar, error)
	Save(ctx context.Context, calendar *domain.Calendar) error
	Del(ctx context.Context, year int64) error
	Get(ctx context.Context, year int64) (*domain.Calendar, error)
}

type calendarService struct {
	repo repository.ContentRepo[model.Calendar]
	l    logger.Logger
}

func NewCalendarService(repo repository.ContentRepo[model.Calendar], l logger.Logger) CalendarService {
	return &calendarService{repo: repo, l: l}
}

func (s *calendarService) GetList(ctx context.Context) ([]domain.Calendar, error) {
	ms, err := s.repo.GetList(ctx)
	if err != nil {
		return nil, errorx.Errorf("获取校历列表失败:%w", err)
	}
	return s.toDomainList(ms), nil
}

func (s *calendarService) Get(ctx context.Context, year int64) (*domain.Calendar, error) {
	m, err := s.repo.Get(ctx, "year", year)
	if err != nil {
		return nil, errorx.Errorf("获取 %d 年校历列表失败:%w", year, err)
	}
	return &domain.Calendar{Year: m.Year, Link: m.Link}, nil
}

func (s *calendarService) Save(ctx context.Context, cal *domain.Calendar) error {

	// 1. 尝试获取现有数据，处理 Upsert 逻辑
	m, err := s.repo.Get(ctx, "year", cal.Year)
	if err != nil {
		if errors.Is(err, repository.ErrRecordNotFound) {
			m = &model.Calendar{Year: cal.Year}
		} else {
			return errorx.Errorf("获取 %d 年校历列表失败:%w", cal.Year, err)
		}
	}

	m.Link = cal.Link

	// 2. 调用 Repo 保存（Repo 内部会自动触发异步缓存刷新）
	if err := s.repo.Save(ctx, m); err != nil {
		return errorx.Errorf("保存 %d 年校历列表失败:%w", cal.Year, err)
	}
	return nil
}

func (s *calendarService) Del(ctx context.Context, year int64) error {
	if err := s.repo.Del(ctx, "year", year); err != nil {
		return errorx.Errorf("删除 %d 年校历列表失败:%w", year, err)
	}
	return nil
}

// 转换逻辑保留在 Service 层，因为它是业务相关的
func (s *calendarService) toDomainList(ms []model.Calendar) []domain.Calendar {
	res := make([]domain.Calendar, 0, len(ms))
	for _, m := range ms {
		res = append(res, domain.Calendar{Year: m.Year, Link: m.Link})
	}
	return res
}
