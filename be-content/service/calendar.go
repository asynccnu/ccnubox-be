package service

import (
	"context"
	"errors"

	"github.com/asynccnu/ccnubox-be/be-content/domain"
	"github.com/asynccnu/ccnubox-be/be-content/repository"
	"github.com/asynccnu/ccnubox-be/be-content/repository/model"
	contentv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/content/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

// 定义 Calendar 相关的错误
var (
	GET_CALENDAR_ERROR = errorx.FormatErrorFunc(contentv1.ErrorGetCalendarError("获取校历失败"))

	DEL_CALENDAR_ERROR = errorx.FormatErrorFunc(contentv1.ErrorDelCalendarError("删除校历失败"))

	SAVE_CALENDAR_ERROR = errorx.FormatErrorFunc(contentv1.ErrorSaveCalendarError("保存校历失败"))
)

// CalendarService 定义接口
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
	return &calendarService{
		repo: repo,
		l:    l,
	}
}

// GetList 获取所有校历列表
func (s *calendarService) GetList(ctx context.Context) ([]domain.Calendar, error) {
	ms, err := s.repo.GetList(ctx)
	if err != nil {
		// 使用 errorx 包装底层错误并关联业务错误码
		return nil, GET_CALENDAR_ERROR(err)
	}
	return s.toDomainList(ms), nil
}

// Get 根据年份获取特定校历
func (s *calendarService) Get(ctx context.Context, year int64) (*domain.Calendar, error) {
	m, err := s.repo.Get(ctx, "year", year)
	if err != nil {
		if errors.Is(err, repository.ErrRecordNotFound) {
			return nil, err
		}
		return nil, GET_CALENDAR_ERROR(errorx.Errorf("查询 %d 年校历失败: %w", year, err))
	}
	return &domain.Calendar{Year: m.Year, Link: m.Link}, nil
}

// Save 保存或更新校历
func (s *calendarService) Save(ctx context.Context, cal *domain.Calendar) error {
	// 1. 尝试获取现有数据，处理 Upsert 逻辑
	m, err := s.repo.Get(ctx, "year", cal.Year)
	if err != nil && !errors.Is(err, repository.ErrRecordNotFound) {
		return SAVE_CALENDAR_ERROR(errorx.Errorf("保存前查询旧校历(year=%d)失败: %w", cal.Year, err))
	}

	// 2. 如果没找到旧数据，初始化新对象
	if m == nil {
		m = &model.Calendar{Year: cal.Year}
	}

	m.Link = cal.Link

	// 3. 调用 Repo 保存
	if err := s.repo.Save(ctx, m); err != nil {
		return SAVE_CALENDAR_ERROR(errorx.Errorf("执行校历(year=%d)保存操作失败: %w", cal.Year, err))
	}
	return nil
}

// Del 删除指定年份校历
func (s *calendarService) Del(ctx context.Context, year int64) error {
	// 业务层防御性校验
	if year <= 0 {
		return DEL_CALENDAR_ERROR(errorx.Errorf("传入了无效的年份 (%d)", year))
	}

	if err := s.repo.Del(ctx, "year", year); err != nil {
		return DEL_CALENDAR_ERROR(errorx.Errorf("删除 %d 年校历失败: %w", year, err))
	}
	return nil
}

// toDomainList 内部模型转换逻辑
func (s *calendarService) toDomainList(ms []model.Calendar) []domain.Calendar {
	res := make([]domain.Calendar, 0, len(ms))
	for _, m := range ms {
		res = append(res, domain.Calendar{
			Year: m.Year,
			Link: m.Link,
		})
	}
	return res
}
