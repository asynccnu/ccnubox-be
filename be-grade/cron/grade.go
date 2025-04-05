package cron

import (
	"context"
	"fmt"
	classlistv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/classlist/v1"
	counterv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/counter/v1"
	feedv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/feed/v1"
	userv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/user/v1"
	"github.com/asynccnu/ccnubox-be/be-grade/pkg/logger"
	"github.com/asynccnu/ccnubox-be/be-grade/service"
	"github.com/spf13/viper"
	"time"
)

type GradeController struct {
	counter      counterv1.CounterServiceClient
	userClient   userv1.UserServiceClient
	feedClient   feedv1.FeedServiceClient
	classlist    classlistv1.ClasserClient
	gradeService service.GradeService
	stopChan     chan struct{}
	cfg          gradeControllerConfig
	l            logger.Logger
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
) *GradeController {
	var cfg gradeControllerConfig
	if err := viper.UnmarshalKey("gradeController", &cfg); err != nil {
		panic(err)
	}

	return &GradeController{
		counter:      counter,
		gradeService: gradeService,
		feedClient:   feedClient,
		classlist:    classlist,
		userClient:   userClient,
		stopChan:     make(chan struct{}),
		cfg:          cfg,
		l:            l,
	}
}

func (c *GradeController) StartCronTask() {
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

	ctx := context.Background()

	resp, err := c.counter.GetCounterLevels(ctx, &counterv1.GetCounterLevelsReq{Label: label})
	if err != nil {
		c.l.Error("获取UserLevels失败", logger.FormatLog("userLevels", err)...)
		return
	}

	for _, studentId := range resp.StudentIds {
		//获取成绩
		grades, err := c.gradeService.GetUpdateScore(ctx, studentId)
		if err != nil {
			c.l.Error("获取成绩失败", logger.FormatLog("feed", err)...)
			return
		}

		//逐个推送
		for _, grade := range grades {
			//获取学生id
			res, err := c.classlist.GetStuIdByJxbId(ctx, &classlistv1.GetStuIdByJxbIdRequest{JxbId: grade.JxbId})
			if err != nil {
				return
			}

			//更改等级到最高级别7
			_, err = c.counter.ChangeCounterLevels(ctx, &counterv1.ChangeCounterLevelsReq{
				StudentIds: res.StuId,
				IsReduce:   false,
				Step:       7,
			})

			if err != nil {
				c.l.Error("更改优先级发生错误", logger.FormatLog("feed", err)...)
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
				c.l.Error("推送错误", logger.FormatLog("feed", err)...)
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
