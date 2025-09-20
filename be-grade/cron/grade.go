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

			//更改等级到最高级别7
			_, err = c.counter.ChangeCounterLevels(ctx, &counterv1.ChangeCounterLevelsReq{
				StudentIds: res.StuId,
				IsReduce:   false,
				Step:       7,
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

		// 获取研究生成绩
		graduateGrades, err := c.gradeService.GetGraduateUpdateScore(ctx, studentId, 0, 0, 0)
		if err != nil {
			c.l.Error("获取研究生成绩失败", logger.Error(err))
			// 不直接 return，继续处理其他学生
			continue
		}

		for _, g := range graduateGrades {
			// 使用教学班ID映射学生ID
			res, err := c.classlist.GetStuIdByJxbId(ctx, &classlistv1.GetStuIdByJxbIdRequest{JxbId: g.ClassID})
			if err != nil {
				c.l.Warn("获取教学班学生ID失败(研究生)", logger.Error(err))
				continue
			}

			// 提升相关学生的优先级
			_, err = c.counter.ChangeCounterLevels(ctx, &counterv1.ChangeCounterLevelsReq{
				StudentIds: res.StuId,
				IsReduce:   false,
				Step:       7,
			})
			if err != nil {
				c.l.Error("更改优先级发生错误(研究生)", logger.Error(err))
				continue
			}

			// 推送研究生成绩更新通知
			_, err = c.feedClient.PublicFeedEvent(ctx, &feedv1.PublicFeedEventReq{
				StudentId: studentId,
				Event: &feedv1.FeedEvent{
					Type:    "grade",
					Title:   "成绩更新提醒(研究生)",
					Content: fmt.Sprintf("您的课程:%s分数更新了,请及时查看", g.ClassName),
				},
			})
			if err != nil {
				c.l.Error("推送错误(研究生)", logger.Error(err))
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
