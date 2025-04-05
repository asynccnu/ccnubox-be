package cron

import (
	"context"
	feedv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/feed/v1"
	"github.com/asynccnu/ccnubox-be/be-calendar/pkg/logger"
	"github.com/asynccnu/ccnubox-be/be-calendar/pkg/lunar"
	"github.com/spf13/viper"
	"time"
)

type HolidayController struct {
	feedClient feedv1.FeedServiceClient
	stopChan   chan struct{}
	cfg        HolidayControllerConfig
	l          logger.Logger
}

type HolidayControllerConfig struct {
	DurationTime int64 `yaml:"durationTime"`
	AdvanceDay   int64 `yaml:"advanceDay"`
}

func NewHolidayController(
	feedClient feedv1.FeedServiceClient,
	l logger.Logger,
) *HolidayController {
	var cfg HolidayControllerConfig
	if err := viper.UnmarshalKey("holidayController", &cfg); err != nil {
		panic(err)
	}
	return &HolidayController{
		feedClient: feedClient,
		stopChan:   make(chan struct{}),
		cfg:        cfg,
		l:          l,
	}
}

func (r *HolidayController) StartCronTask() {
	go func() {
		ticker := time.NewTicker(time.Duration(r.cfg.DurationTime) * time.Hour)
		for {
			select {
			case <-ticker.C:
				err := r.publishMSG()
				if err != nil {
					r.l.Error("推送消息失败!:", logger.FormatLog("cron", err)...)
				}

			case <-r.stopChan:
				ticker.Stop()
				return
			}
		}
	}() //定时控制器

}

func (r *HolidayController) publishMSG() error {
	//由于没有使用注册为路由这里手动写的上下文,每次提前四天进行提醒
	holiday := lunar.IsHoliday(time.Now().Add(time.Duration(r.cfg.AdvanceDay) * 24 * time.Hour))
	if holiday == "" {
		return nil
	}

	ctx := context.Background()
	//发送给全体成员
	_, err := r.feedClient.PublicFeedEvent(ctx, &feedv1.PublicFeedEventReq{
		IsAll: true,
		Event: &feedv1.FeedEvent{
			Type:    "holiday",
			Title:   "假期临近提醒",
			Content: holiday + "假期临近,请及时查看放假通知及调休安排",
		},
	})
	if err != nil {
		return nil
	}

	return err
}
