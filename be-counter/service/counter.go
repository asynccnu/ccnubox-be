package service

import (
	"context"
	"fmt"
	"github.com/asynccnu/ccnubox-be/be-counter/domain"
	"github.com/asynccnu/ccnubox-be/be-counter/pkg/logger"
	"github.com/asynccnu/ccnubox-be/be-counter/repository/cache"
	"github.com/spf13/viper"
)

type CounterService interface {
	AddCounter(ctx context.Context, StudentId string) error
	GetCounterLevels(ctx context.Context, label string) (StudentIds []string, err error)
	ChangeCounterLevels(ctx context.Context, req domain.ChangeCounterLevels) error
	ClearCounterLevels(ctx context.Context) error
}

type CachedCounterService struct {
	cache  cache.CounterCache //此处不做任何持久化,所有的数据都存到Redis中
	l      logger.Logger
	config CounterConfig
}

type CounterConfig struct {
	Low           int64
	MStudentIddle int64
	High          int64
	Step          int64 //一次更改的大小
}

func NewCachedCounterService(cache cache.CounterCache, l logger.Logger) CounterService {
	var config CounterConfig
	err := viper.UnmarshalKey("countLevel", &config)
	if err != nil {
		return nil
	}
	return &CachedCounterService{cache: cache, l: l, config: config}
}

func (repo *CachedCounterService) AddCounter(ctx context.Context, StudentId string) error {
	var count int64
	count, err := repo.cache.GetCounterByStudentId(ctx, StudentId)
	if err != nil {
		count = 0
	}

	err = repo.cache.SetCounterByStudentId(ctx, StudentId, count+1)
	if err != nil {
		return err
	}

	return nil
}

// 建议优化 TODO 使用数值的方式过于magic不太适合微服务通信
func (repo *CachedCounterService) GetCounterLevels(ctx context.Context, label string) ([]string, error) {
	// 获取所有 Counter
	counts, err := repo.cache.GetAllCounter(ctx)
	if err != nil {
		return nil, err
	}

	// 预先计算阈值
	lowThreshold := repo.config.Low
	mStudentIdThreshold := repo.config.MStudentIddle
	highThreshold := repo.config.High

	// 预先分配一个大致的容量，避免多次扩容
	StudentIds := make([]string, 0, len(counts))

	// 判断 label 的合法性
	switch label {
	case "low":
		for _, count := range counts {
			if count.Count >= lowThreshold && count.Count < mStudentIdThreshold {
				StudentIds = append(StudentIds, count.StudentId)
			}
		}
	case "middle":
		for _, count := range counts {
			if count.Count >= mStudentIdThreshold && count.Count < highThreshold {
				StudentIds = append(StudentIds, count.StudentId)
			}
		}
	case "high":
		for _, count := range counts {
			if count.Count >= highThreshold {
				StudentIds = append(StudentIds, count.StudentId)
			}
		}
	default:
		return nil, fmt.Errorf("invalStudentId label: %s", label)
	}

	return StudentIds, nil
}

func (repo *CachedCounterService) ChangeCounterLevels(ctx context.Context, req domain.ChangeCounterLevels) error {
	counts, err := repo.cache.GetCounters(ctx, req.StudentIds)
	if err != nil {
		return err
	}
	//设定长度乘上轮询的步数
	step := repo.config.Step * req.Steps
	//根据是否是降低来进行不同的操作
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

	err = repo.cache.SetCounters(ctx, counts)
	if err != nil {
		return err
	}
	return nil
}

func (repo *CachedCounterService) ClearCounterLevels(ctx context.Context) error {
	return repo.cache.CleanZeroCounter(ctx)
}
