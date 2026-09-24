package cron

import (
	"context"
	"strings"
	"sync"
	"time"

	feedv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feed/v1"

	"github.com/asynccnu/ccnubox-be/be-feed/conf"
	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	"github.com/asynccnu/ccnubox-be/be-feed/pkg/lunar"
	"github.com/asynccnu/ccnubox-be/be-feed/service"

	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/asynccnu/ccnubox-be/common/pkg/saramax"
)

const HOLIDAY_EVENT_URL = "ccnubox://calendar"

type HolidayController struct {
	svcFeed  service.FeedEventService
	stopChan chan struct{}
	cfg      *conf.HolidayControllerConfig
	l        logger.Logger
	stopOnce sync.Once
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

func (r *HolidayController) StopCronTask() {
	r.stopOnce.Do(func() {
		r.cancel()
		close(r.stopChan)
	})
	r.wg.Wait()
}

func NewHolidayController(
	svcFeed service.FeedEventService,
	l logger.Logger,
	cfg *conf.ServerConf,
) *HolidayController {

	ctx, cancel := context.WithCancel(context.Background())
	return &HolidayController{
		ctx:      ctx,
		cancel:   cancel,
		svcFeed:  svcFeed,
		stopChan: make(chan struct{}),
		cfg:      cfg.HolidayController,
		l:        l,
	}
}

func (r *HolidayController) StartCronTask() error {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		ticker := time.NewTicker(time.Duration(r.cfg.DurationTime) * time.Hour)
		for {
			select {
			case <-ticker.C:
				err := r.publishMSG()
				if err != nil {
					r.l.Error(saramax.LogKeySendFailed+" 推送假日提醒失败，未推送的收件人需用相同 dedupe_key 重跑", logger.Error(err))
				}

			case <-r.stopChan:
				ticker.Stop()
				return
			}
		}
	}() //定时控制器
	return nil
}

func (r *HolidayController) publishMSG() error {
	//由于没有使用注册为路由这里手动写的上下文,每次提前四天进行提醒
	holidayDate := time.Now().Add(time.Duration(r.cfg.AdvanceDay) * 24 * time.Hour)
	holiday := lunar.IsHoliday(holidayDate)
	if holiday == "" {
		return nil
	}

	ctx := r.ctx
	//发送给全体成员
	_, err := r.svcFeed.PublicFeedEvent(ctx, true, domain.FeedEvent{
		DedupeKey:    "holiday:" + holidayDate.Format("2006-01-02") + ":" + holiday,
		Type:         strings.ToLower(feedv1.FeedEventType_HOLIDAY.String()),
		Title:        "假期临近提醒",
		Content:      holiday + "假期临近,请及时查看放假通知及调休安排",
		Url:          HOLIDAY_EVENT_URL,
		ExtendFields: map[string]string{"url": HOLIDAY_EVENT_URL},
	})
	return err
}
