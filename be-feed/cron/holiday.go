package cron

import (
	"context"
	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	"github.com/asynccnu/ccnubox-be/be-feed/pkg/lunar"
	"github.com/asynccnu/ccnubox-be/be-feed/service"
	"time"

	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/spf13/viper"
)

type HolidayController struct {
	svcFeed  service.FeedEventService
	stopChan chan struct{}
	cfg      HolidayControllerConfig
	l        logger.Logger
}

type HolidayControllerConfig struct {
	DurationTime int64 `yaml:"durationTime"`
	AdvanceDay   int64 `yaml:"advanceDay"`
}

func NewHolidayController(
	svcFeed service.FeedEventService,
	l logger.Logger,
) *HolidayController {
	var cfg HolidayControllerConfig
	if err := viper.UnmarshalKey("holidayController", &cfg); err != nil {
		panic(err)
	}
	return &HolidayController{
		svcFeed:  svcFeed,
		stopChan: make(chan struct{}),
		cfg:      cfg,
		l:        l,
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
					r.l.Error("推送消息失败!:", logger.Error(err))
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
	err := r.svcFeed.PublicFeedEvent(ctx, true, domain.FeedEvent{
		Type:    "holiday",
		Title:   "假期临近提醒",
		Content: holiday + "假期临近,请及时查看放假通知及调休安排",
	})
	if err != nil {
		return nil
	}

	return err
}
