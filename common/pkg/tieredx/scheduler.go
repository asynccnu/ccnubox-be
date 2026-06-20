package tieredx

import (
	"context"
	"fmt"
	"sync"
	"time"

	counterv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/counter/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/asynccnu/ccnubox-be/common/tool"
	"github.com/go-redsync/redsync/v4"
	"github.com/panjf2000/ants/v2"
	"golang.org/x/time/rate"
)

// 使用协程池
const (
	qps      = 20
	PoolSize = 20
)

// RefreshHandler 此接口用来封装对单个学号的刷新逻辑
type RefreshHandler interface {
	Refresh(ctx context.Context, studentId string) error
}

type TieredConfig struct {
	Low    int
	Middle int
	High   int
}

type Option func(*TieredScheduler)

// WithRedsync 注入分布式锁，防止多实例重复执行。
func WithRedsync(rs *redsync.Redsync) Option {
	return func(s *TieredScheduler) {
		s.rs = rs
	}
}

// TieredScheduler 通用分层刷新调度器。
type TieredScheduler struct {
	cfg      TieredConfig
	handler  RefreshHandler
	counter  counterv1.CounterServiceClient
	l        logger.Logger
	rs       *redsync.Redsync
	stopChan chan struct{}
	pool     *ants.Pool
}

// NewTieredScheduler 构造调度器。
func NewTieredScheduler(
	cfg TieredConfig,
	handler RefreshHandler,
	counter counterv1.CounterServiceClient,
	l logger.Logger,
	opts ...Option,
) *TieredScheduler {
	pool, _ := ants.NewPool(
		PoolSize,
		ants.WithExpiryDuration(60*time.Second),
		ants.WithNonblocking(false), //阻塞模式
	)
	s := &TieredScheduler{
		cfg:      cfg,
		handler:  handler,
		counter:  counter,
		l:        l,
		stopChan: make(chan struct{}),
		pool:     pool,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Start 启动三个档位的 ticker goroutine。
func (s *TieredScheduler) Start() {
	s.startTicker(time.Duration(s.cfg.Low)*time.Minute, "low")
	s.startTicker(time.Duration(s.cfg.Middle)*time.Minute, "middle")
	s.startTicker(time.Duration(s.cfg.High)*time.Minute, "high")
}

func (s *TieredScheduler) startTicker(interval time.Duration, label string) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.runRefresh(label)
			case <-s.stopChan:
				return
			}
		}
	}()
}

func (s *TieredScheduler) runRefresh(label string) {
	// 若配置了分布式锁，先尝试获取
	if s.rs != nil {
		lockKey := fmt.Sprintf("tiered-refresh:%s", label)
		lock := s.rs.NewMutex(lockKey, redsync.WithTries(1))
		if err := lock.Lock(); err != nil {
			s.l.Warnf("获取分布式锁失败:%v", err)
			return
		}
		defer lock.Unlock()
	}

	ctx := context.Background()

	resp, err := s.counter.GetCounterLevels(ctx, &counterv1.GetCounterLevelsReq{
		Label: label,
	})
	if err != nil {
		s.l.Errorf("获取UserLevels失败 label:%s error:%v ", label, err)
		return
	}
	stuIDs := resp.GetStudentIds()
	if len(stuIDs) == 0 {
		return
	}

	limiter := rate.NewLimiter(rate.Limit(qps), qps)

	var wg sync.WaitGroup

	for _, stuID := range stuIDs {
		stuID := stuID
		//跳过毕业学生
		if tool.IsGraduated(stuID) {
			continue
		}

		wg.Add(1)
		err = s.pool.Submit(func() {
			defer func() {
				wg.Done()
				//防止一个任务panic整个协程结束
				if r := recover(); r != nil {
					s.l.Errorf("student:%s task submit panic: %v", stuID, r)
				}
			}()

			//尝试获取令牌：
			err := limiter.Wait(ctx)
			if err != nil {
				s.l.Warnf("limiter wait canceled: %v", err)
				return
			}

			ct, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			err = s.handler.Refresh(ct, stuID)
			if err != nil {
				s.l.Errorf("handler refresh error:%v", err)
				return
			}

		})
		if err != nil {
			wg.Done()
			s.l.Errorf("pool submit task error:%v", err)
			continue
		}
	}

	wg.Wait()

	_, err = s.counter.ChangeCounterLevels(ctx, &counterv1.ChangeCounterLevelsReq{
		StudentIds: resp.StudentIds,
		IsReduce:   true,
		Step:       7,
	})
	if err != nil {
		s.l.Errorf("降级失败:%v", err)
	}

}

func (s *TieredScheduler) Stop() {
	close(s.stopChan)
	s.pool.Release()
}
