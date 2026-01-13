package cron

import (
	"context"
	"time"

	"github.com/asynccnu/ccnubox-be/be-feed/conf"
	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	"github.com/asynccnu/ccnubox-be/be-feed/pkg/lunar"
	"github.com/asynccnu/ccnubox-be/be-feed/service"

	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

type HolidayController struct {
	svcFeed  service.FeedEventService
	stopChan chan struct{}
	cfg      *conf.HolidayControllerConfig
	l        logger.Logger
}

func NewHolidayController(
	svcFeed service.FeedEventService,
	l logger.Logger,
	cfg *conf.ServerConf,
) *HolidayController {

	return &HolidayController{
		svcFeed:  svcFeed,
		stopChan: make(chan struct{}),
		cfg:      cfg.HolidayController,
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
