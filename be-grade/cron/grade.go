package cron

import (
	"context"
	"fmt"
	"time"

	classlistv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/classlist/v1"
	counterv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/counter/v1"
	feedv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/feed/v1"
	userv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/user/v1"
	"github.com/asynccnu/ccnubox-be/be-grade/pkg/logger"
	"github.com/asynccnu/ccnubox-be/be-grade/service"
	"github.com/go-redsync/redsync/v4"
	"github.com/spf13/viper"
)

type GradeController struct {
	counter      counterv1.CounterServiceClient
	userClient   userv1.UserServiceClient
	feedClient   feedv1.FeedServiceClient
	classlist    classlistv1.ClasserClient
	gradeService service.GradeService
	rankService  service.RankService
	stopChan     chan struct{}
	cfg          gradeControllerConfig
	l            logger.Logger
	muRedis      *redsync.Redsync
}

type gradeControllerConfig struct {
	Low    int64 `yaml:"low"`
	Middle int64 `yaml:"middle"`
	High   int64 `yaml:"high"`
}

func NewGradeController(
	l logger.Logger,
	counter counterv1.CounterServiceClient,
	userClient userv1.UserServiceClient,
	feedClient feedv1.FeedServiceClient,
	classlist classlistv1.ClasserClient,
	gradeService service.GradeService,
	rankService service.RankService,
	muRedis *redsync.Redsync,
) *GradeController {
	var cfg gradeControllerConfig
	if err := viper.UnmarshalKey("gradeController", &cfg); err != nil {
		panic(err)
	}

	return &GradeController{
		counter:      counter,
		gradeService: gradeService,
		rankService:  rankService,
		feedClient:   feedClient,
		classlist:    classlist,
		userClient:   userClient,
		stopChan:     make(chan struct{}),
		cfg:          cfg,
		l:            l,
		muRedis:      muRedis,
	}
}

func (c *GradeController) StartCronTask() {
	// 开启rank的定时任务，具体看rank.go
	c.StartRankCronTask()

	go func() {
		lowTicker := time.NewTicker(time.Duration(c.cfg.Low) * time.Minute)
		middleTicker := time.NewTicker(time.Duration(c.cfg.Middle) * time.Minute)
		highTicker := time.NewTicker(time.Duration(c.cfg.High) * time.Minute)
		for {
			select {
			case <-lowTicker.C:
				c.publishMSG("low")

			case <-middleTicker.C:
				c.publishMSG("middle")

			case <-highTicker.C:
				c.publishMSG("high")

			case <-c.stopChan:
				lowTicker.Stop()
				middleTicker.Stop()
				highTicker.Stop()
				return
			}
		}
	}() //定时控制器

}

func (c *GradeController) publishMSG(label string) {
	lock := c.muRedis.NewMutex("推送成绩: "+label, redsync.WithTries(1))

	err := lock.Lock()
	if err != nil {
		// 防止不是竞争锁失败，而是别的问题导致的出错
		c.l.Warn("获取分布式锁失败", logger.Error(err))
		return
	}
	defer lock.Unlock()

	ctx := context.Background()

	resp, err := c.counter.GetCounterLevels(ctx, &counterv1.GetCounterLevelsReq{Label: label})
	if err != nil {
		c.l.Error("获取UserLevels失败", logger.Error(err))
		return
	}

	for _, studentId := range resp.StudentIds {
		//获取本科生成绩
		grades, err := c.gradeService.GetUpdateScore(ctx, studentId)
		if err != nil {
			c.l.Error("获取成绩失败", logger.Error(err))
			return
		}

		//逐个推送(本科生)
		for _, grade := range grades {
			//获取学生id
			res, err := c.classlist.GetStuIdByJxbId(ctx, &classlistv1.GetStuIdByJxbIdRequest{JxbId: grade.JxbId})
			if err != nil {
				return
			}

			//更改等级到最高级别
			_, err = c.counter.ChangeCounterLevels(ctx, &counterv1.ChangeCounterLevelsReq{
				StudentIds: res.StuId,
				IsReduce:   false,
				Step:       int64(counterv1.CounterLevel_LEVEL_THERE),
			})

			if err != nil {
				c.l.Error("更改优先级发生错误", logger.Error(err))
				return
			}

			//推送
			_, err = c.feedClient.PublicFeedEvent(ctx, &feedv1.PublicFeedEventReq{
				StudentId: studentId,
				Event: &feedv1.FeedEvent{
					Type:    "grade",
					Title:   "成绩更新提醒",
					Content: fmt.Sprintf("您的课程:%s分数更新了,请及时查看", grade.Kcmc),
				},
			})
			if err != nil {
				c.l.Error("推送错误", logger.Error(err))
			}
		}
	}

	//更改已经完成的studentId等级到最低等级
	_, err = c.counter.ChangeCounterLevels(ctx, &counterv1.ChangeCounterLevelsReq{
		StudentIds: resp.StudentIds,
		IsReduce:   false,
		Step:       7,
	})

	return
}
