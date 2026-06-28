package service

import (
	"context"
	"fmt"

	"github.com/asynccnu/ccnubox-be/be-counter/repository/cache"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

type CounterService interface {
	AddCounter(ctx context.Context, studentId string) error
	DecayCounter(ctx context.Context, studentIds []string) error
	BoostScores(ctx context.Context, studentIds []string) error
	RebuildCounter(ctx context.Context) error
	GetCounterLevels(ctx context.Context, label string) ([]string, error)
}

type CachedCounterService struct {
	cache cache.CounterCache
	l     logger.Logger
}

func NewCachedCounterService(cache cache.CounterCache, l logger.Logger) CounterService {
	return &CachedCounterService{cache: cache, l: l}
}

func (s *CachedCounterService) AddCounter(ctx context.Context, studentId string) error {
	_, err := s.cache.AddCounter(ctx, studentId)
	if err != nil {
		return errorx.Errorf("service: add counter failed: %w", err)
	}
	return nil
}

func (s *CachedCounterService) RebuildCounter(ctx context.Context) error {
	if err := s.cache.RebuildCounter(ctx); err != nil {
		return errorx.Errorf("service: rebuild failed: %w", err)
	}
	return nil
}

func (s *CachedCounterService) GetCounterLevels(ctx context.Context, label string) ([]string, error) {
	total, err := s.cache.GetCounterCount(ctx)
	if err != nil {
		return nil, err
	}
	third := total / 3
	switch label {
	case "low":
		res, err := s.cache.GetCounterByRank(ctx, 0, third)
		if err != nil {
			return nil, errorx.Errorf("service: get counter low level failed: %w", err)
		}
		return res, nil
	case "middle":
		res, err := s.cache.GetCounterByRank(ctx, third+1, third*2)
		if err != nil {
			return nil, errorx.Errorf("service: get counter middle level failed: %w", err)
		}
		return res, nil
	case "high":
		res, err := s.cache.GetCounterByRank(ctx, third*2+1, third*3)
		if err != nil {
			return nil, errorx.Errorf("service: get counter high level failed: %w", err)
		}
		return res, nil
	}

	return nil, fmt.Errorf("service: invalid label: %s", label)
}

func (s *CachedCounterService) DecayCounter(ctx context.Context, studentIds []string) error {
	if err := s.cache.DecayCounter(ctx, studentIds); err != nil {
		return errorx.Errorf("service: decay failed: %w", err)
	}
	return nil
}

func (s *CachedCounterService) BoostScores(ctx context.Context, studentIds []string) error {
	if err := s.cache.BoostScores(ctx, studentIds); err != nil {
		return errorx.Errorf("service: boost failed: %w", err)
	}
	return nil
}
