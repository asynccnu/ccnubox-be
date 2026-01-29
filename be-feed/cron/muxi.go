package cron

import (
	"context"
	feedv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feed/v1"
	"strings"
	"time"

	"github.com/asynccnu/ccnubox-be/be-feed/conf"
	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	"github.com/asynccnu/ccnubox-be/be-feed/service"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

type MuxiController struct {
	muxi         service.MuxiOfficialMSGService
	push         service.PushService
	feed         service.FeedEventService
	durationTime time.Duration
	stopChan     chan struct{}
	l            logger.Logger
}

func NewMuxiController(
	muxi service.MuxiOfficialMSGService,
	feed service.FeedEventService,
	push service.PushService,
	l logger.Logger,
	cfg *conf.ServerConf,
) *MuxiController {
	return &MuxiController{
		muxi:         muxi,
		push:         push,
		feed:         feed,
		durationTime: time.Duration(cfg.MuxiController.DurationTime) * time.Second,
		stopChan:     make(chan struct{}),
		l:            l,
	}
}

func (c *MuxiController) StartCronTask() {
	go func() {
		ticker := time.NewTicker(c.durationTime)

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
	msgs, err := c.muxi.GetToBePublicOfficialMSG(ctx, true)
	if err != nil {
		c.l.Warn("获取木犀消息失败!", logger.Error(err))
		return
	}
	if len(msgs) == 0 {
		return
	}

	for _, msg := range msgs {
		//发布消息给全体成员
		err = c.feed.PublicFeedEvent(ctx, true, domain.FeedEvent{
			Type:         strings.ToLower(feedv1.FeedEventType_MUXI.String()),
			Title:        msg.Title,
			Content:      msg.Content,
			ExtendFields: msg.ExtendFields,
		})

		if err != nil {
			c.l.Warn("消息推送失败!", logger.Error(err))
			return
		}
	}

	return
}
