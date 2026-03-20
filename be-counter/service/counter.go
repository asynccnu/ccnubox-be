package service

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-counter/conf"
	"github.com/asynccnu/ccnubox-be/be-counter/domain"
	"github.com/asynccnu/ccnubox-be/be-counter/repository/cache"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

type CounterService interface {
	AddCounter(ctx context.Context, StudentId string) error
	GetCounterLevels(ctx context.Context, label string) (StudentIds []string, err error)
	ChangeCounterLevels(ctx context.Context, req domain.ChangeCounterLevels) error
	ClearCounterLevels(ctx context.Context) error
}

type CachedCounterService struct {
	cache  cache.CounterCache
	l      logger.Logger
	config *conf.CountLevelConfig
}

func NewCachedCounterService(cache cache.CounterCache, l logger.Logger, cfg *conf.ServerConf) CounterService {
	return &CachedCounterService{
		cache:  cache,
		l:      l,
		config: cfg.CountLevel,
	}
}

func (s *CachedCounterService) AddCounter(ctx context.Context, StudentId string) error {
	// 获取当前计数
	count, err := s.cache.GetCounterByStudentId(ctx, StudentId)
	if err != nil {
		// 注意：如果 cache 层处理了 redis.Nil 并返回 0, nil，则这里不会报错
		// 如果 cache 层报错，说明是 Redis 连接等问题
		return errorx.Errorf("service: failed to get current counter for student %s: %w", StudentId, err)
	}

	// 增加计数
	err = s.cache.SetCounterByStudentId(ctx, StudentId, count+1)
	if err != nil {
		return errorx.Errorf("service: failed to increment counter for student %s: %w", StudentId, err)
	}

	return nil
}

func (s *CachedCounterService) GetCounterLevels(ctx context.Context, label string) ([]string, error) {
	counts, err := s.cache.GetAllCounter(ctx)
	if err != nil {
		return nil, errorx.Errorf("service: failed to fetch all counters from cache: %w", err)
	}

	lowThreshold := s.config.Low
	middleThreshold := s.config.Middle
	highThreshold := s.config.High

	StudentIds := make([]string, 0, len(counts))

	switch label {
	case "low":
		for _, c := range counts {
			if c.Count >= lowThreshold && c.Count < middleThreshold {
				StudentIds = append(StudentIds, c.StudentId)
			}
		}
	case "middle":
		for _, c := range counts {
			if c.Count >= middleThreshold && c.Count < highThreshold {
				StudentIds = append(StudentIds, c.StudentId)
			}
		}
	case "high":
		for _, c := range counts {
			if c.Count >= highThreshold {
				StudentIds = append(StudentIds, c.StudentId)
			}
		}
	default:
		// 参数校验错误，使用 errorx 封装
		return nil, errorx.Errorf("service: invalid level label: %s", label)
	}

	return StudentIds, nil
}

func (s *CachedCounterService) ChangeCounterLevels(ctx context.Context, req domain.ChangeCounterLevels) error {
	counts, err := s.cache.GetCounters(ctx, req.StudentIds)
	if err != nil {
		return errorx.Errorf("service: failed to get counters for batch update: %w", err)
	}

	step := s.config.Step * req.Steps

	if req.IsReduce {
		for i := range counts {
			if counts[i].Count >= step {
				counts[i].Count -= step
			} else {
				counts[i].Count = 0
			}
		}
	} else {
		for i := range counts {
			counts[i].Count += step
		}
	}

	err = s.cache.SetCounters(ctx, counts)
	if err != nil {
		return errorx.Errorf("service: failed to save updated counters: %w", err)
	}
	return nil
}

func (s *CachedCounterService) ClearCounterLevels(ctx context.Context) error {
	err := s.cache.CleanZeroCounter(ctx)
	if err != nil {
		return errorx.Errorf("service: failed to clear zero counters: %w", err)
	}
	return nil
}
