package cron

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/asynccnu/ccnubox-be/be-classlist/internal/biz"
	contentv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/content/v1"
	counterv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/counter/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/asynccnu/ccnubox-be/common/tool"
	"github.com/go-kratos/kratos/v2/log"
	"golang.org/x/time/rate"
)

type ClassListController struct {
	counter      counterv1.CounterServiceClient
	content      contentv1.ContentServiceClient
	classUsecase *biz.ClassUsecase
	l            logger.Logger
	stopChan     chan struct{}
	mu           *sync.Mutex
}

func NewClassListController(counter counterv1.CounterServiceClient, classUsecase *biz.ClassUsecase, content contentv1.ContentServiceClient, l logger.Logger) Cron {
	return &ClassListController{
		counter:      counter,
		content:      content,
		classUsecase: classUsecase,
		l:            l,
		stopChan:     make(chan struct{}),
		mu:           &sync.Mutex{},
	}
}

func (c *ClassListController) StartCronTask() {
	go func() {
		lowTicker := time.NewTicker(10 * 24 * time.Hour)
		middleTicker := time.NewTicker(7 * 24 * time.Hour)
		highTicker := time.NewTicker(4 * 24 * time.Hour)

		for {
			select {
			case <-lowTicker.C:
				c.pullClassListTask("low")
			case <-middleTicker.C:
				c.pullClassListTask("middle")
			case <-highTicker.C:
				c.pullClassListTask("high")

			case <-c.stopChan:
				lowTicker.Stop()
				middleTicker.Stop()
				highTicker.Stop()
				return
			}
		}
	}()
}

func (c *ClassListController) pullClassListTask(label string) {
	ctx := context.Background()
	//拉取对应等级的学号
	resp, err := c.counter.GetCounterLevels(ctx, &counterv1.GetCounterLevelsReq{
		ServiceType: counterv1.ServiceType_CLASSLIST,
		Label:       label,
	})

	if err != nil {
		c.l.Error("获取UserLevels失败", logger.Error(err))
		return
	}

	stuIDs := resp.GetStudentIds()

	const (
		pageSize        = 200
		workerCount     = 16
		qps             = 20
		progressLogStep = 200
	)

	// 使用官方令牌桶限流
	limiter := rate.NewLimiter(rate.Limit(qps), qps)

	jobs := make(chan string, 1000)

	var (
		wg        sync.WaitGroup
		processed atomic.Int64
	)

	c.l.Infof("PullClassListTask started pageSize=%d workerCount=%d qps=%d", pageSize, workerCount, qps)

	//从content服务中获取当前最近学期
	res, err := c.content.GetSemester(ctx, &contentv1.GetSemesterRequest{})
	if err != nil {
		c.l.Errorf("获取当前学期错误:%v", err)
		return
	}

	//把学期字符串解析成year和semester
	strs := strings.Split(res.Semester, "-")
	year := strs[0]
	semester := strs[1]

	//协程池
	for i := 0; i < workerCount; i++ {
		workID := i + 1
		wg.Add(1)

		go func(id int) {
			defer func() {
				if r := recover(); r != nil {
					log.Errorf("worker-%d panic: %v", id, r)
				}
				wg.Done()
				log.Infof("worker-%d stopped", id)
			}()
			log.Infof("worker-%d started", id)

			for stuID := range jobs {
				if tool.IsGraduated(stuID) {
					// 跳过已经毕业的学生
					log.Infof("worker-%d skipping graduated student %s", id, stuID)
					continue
				}

				//等待令牌
				if err := limiter.Wait(ctx); err != nil {
					log.Warnf("worker-%d limiter wait canceled: %v", id, err)
					continue
				}

				log.Infof("worker-%d processing student %s", id, stuID)

				//单任务超时
				ct, cancel := context.WithTimeout(ctx, 10*time.Second)
				//GetClasses的目的是强制爬虫刷新，写入数据库
				_, _, err := c.classUsecase.GetClasses(ct, stuID, year, semester, true)
				cancel()
				if err != nil {
					c.l.Errorf("刷新课程信息失败：%v", err)
				}

				count := processed.Add(1)
				if count%progressLogStep == 0 {
					log.Infof("processed %d students so far", count)
				}

			}
		}(workID)
	}

	//把studentID写入job
	for _, stuID := range stuIDs {
		jobs <- stuID
	}

	close(jobs)
	wg.Wait()

	//把已经推送过的等级降到最低（防止高频率重复刷新）
	_, err = c.counter.ChangeCounterLevels(ctx, &counterv1.ChangeCounterLevelsReq{
		StudentIds:  resp.StudentIds,
		IsReduce:    true,
		Step:        7,
		ServiceType: counterv1.ServiceType_GRADE,
	})

	total := processed.Load()
	log.Infof("Finished PullClassListTask processed=%d", total)
}
