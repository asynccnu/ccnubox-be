package cron

import (
	"context"
	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	"github.com/asynccnu/ccnubox-be/be-feed/pkg/logger"
	"github.com/asynccnu/ccnubox-be/be-feed/service"
	"github.com/spf13/viper"
	"time"
)

type MuxiController struct {
	muxi     service.MuxiOfficialMSGService
	push     service.PushService
	feed     service.FeedEventService
	cfg      muxiControllerConfig
	stopChan chan struct{}
	l        logger.Logger
}

type muxiControllerConfig struct {
	DurationTime int64 `yaml:"durationTime"`
}

func NewMuxiController(
	muxi service.MuxiOfficialMSGService,
	feed service.FeedEventService,
	push service.PushService,
	l logger.Logger,
) *MuxiController {

	var cfg muxiControllerConfig

	if err := viper.UnmarshalKey("muxiController", &cfg); err != nil {
		panic(err)
	}

	return &MuxiController{
		muxi:     muxi,
		push:     push,
		feed:     feed,
		cfg:      cfg,
		stopChan: make(chan struct{}),
		l:        l,
	}
}

func (c *MuxiController) StartCronTask() {
	go func() {
		ticker := time.NewTicker(time.Duration(c.cfg.DurationTime) * time.Second)

		for {
			select {
			case <-ticker.C:
				c.publicMuxiFeed()
			case <-c.stopChan:
				ticker.Stop()

				return
			}
		}
	}() //定时控制器

}

func (c *MuxiController) publicMuxiFeed() {
	ctx := context.Background()
	//获取feed列表
	msgs, err := c.muxi.GetToBePublicOfficialMSG(ctx)
	if err != nil {
		c.l.Warn("获取木犀消息失败!", logger.FormatLog("cache", err)...)
		return
	}
	for _, msg := range msgs {
		if msg.PublicTime <= time.Now().Unix() {
			err = c.muxi.StopMuxiOfficialMSG(ctx, msg.Id)
			if err != nil {
				return
			}
			//发布消息给全体成员
			err := c.feed.PublicFeedEvent(ctx, true, domain.FeedEvent{
				Type:         "muxi",
				Title:        msg.Title,
				Content:      msg.Content,
				ExtendFields: msg.ExtendFields,
			})
			if err != nil {
				c.l.Warn("消息推送失败!", logger.FormatLog("cache", err)...)
				return
			}
		}
	}

	return
}
