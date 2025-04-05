package cron

import (
	"context"
	"fmt"
	feedv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/feed/v1"
	"github.com/asynccnu/ccnubox-be/be-elecprice/pkg/logger"
	"github.com/asynccnu/ccnubox-be/be-elecprice/service"
	"github.com/spf13/viper"
	"time"
)

type ElecpriceController struct {
	feedClient      feedv1.FeedServiceClient
	elecpriceSerice service.ElecpriceService
	stopChan        chan struct{}
	cfg             ElecpriceControllerConfig
	l               logger.Logger
}

type ElecpriceControllerConfig struct {
	DurationTime int64 `yaml:"durationTime"`
}

func NewElecpriceController(
	feedClient feedv1.FeedServiceClient,
	elecpriceSerice service.ElecpriceService,
	l logger.Logger,
) *ElecpriceController {
	var cfg ElecpriceControllerConfig
	if err := viper.UnmarshalKey("elecpriceController", &cfg); err != nil {
		panic(err)
	}
	return &ElecpriceController{
		feedClient:      feedClient,
		elecpriceSerice: elecpriceSerice,
		stopChan:        make(chan struct{}),
		cfg:             cfg,
		l:               l,
	}
}

func (r *ElecpriceController) StartCronTask() {
	go func() {
		ticker := time.NewTicker(time.Duration(r.cfg.DurationTime) * time.Hour)
		for {
			select {
			case <-ticker.C:
				err := r.publishMSG()
				r.l.Error("推送消息失败!:", logger.FormatLog("cron", err)...)

			case <-r.stopChan:
				ticker.Stop()
				return
			}
		}
	}() //定时控制器

}

func (r *ElecpriceController) publishMSG() error {
	//由于没有使用注册为路由这里手动写的上下文,每次提前四天进行提醒

	ctx := context.Background()
	msgs, err := r.elecpriceSerice.GetTobePushMSG(ctx)
	if err != nil {
		return err
	}
	for i := range msgs {
		if msgs[i].Remain != nil {

			//发送给全体成员
			_, err = r.feedClient.PublicFeedEvent(ctx, &feedv1.PublicFeedEventReq{
				StudentId: msgs[i].StudentId,
				Event: &feedv1.FeedEvent{
					Type:    "energy",
					Title:   "电费不足提醒",
					Content: fmt.Sprintf("您的房间%s当前的电费为:%s,低于设置阈值,请及时充费", *(msgs[i].RoomName), *(msgs[i].Remain)),
				},
			})
		}

	}

	return err
}
