package cron

import (
	"context"
	feedv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feed/v1"
	"strings"
	"sync"
	"time"

	"github.com/asynccnu/ccnubox-be/be-feed/conf"
	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	"github.com/asynccnu/ccnubox-be/be-feed/service"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/asynccnu/ccnubox-be/common/pkg/saramax"
)

type MuxiController struct {
	muxi         service.MuxiOfficialMSGService
	push         service.PushService
	feed         service.FeedEventService
	durationTime time.Duration
	stopChan     chan struct{}
	l            logger.Logger
	stopOnce     sync.Once
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

func (c *MuxiController) StopCronTask() {
	c.stopOnce.Do(func() {
		c.cancel()
		close(c.stopChan)
	})
	c.wg.Wait()
}

func NewMuxiController(
	muxi service.MuxiOfficialMSGService,
	feed service.FeedEventService,
	push service.PushService,
	l logger.Logger,
	cfg *conf.ServerConf,
) *MuxiController {
	ctx, cancel := context.WithCancel(context.Background())
	return &MuxiController{
		ctx:          ctx,
		cancel:       cancel,
		muxi:         muxi,
		push:         push,
		feed:         feed,
		durationTime: time.Duration(cfg.MuxiController.DurationTime) * time.Second,
		stopChan:     make(chan struct{}),
		l:            l,
	}
}

func (c *MuxiController) StartCronTask() error {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
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
	return nil
}

func (c *MuxiController) publicMuxiFeed() {
	ctx := c.ctx
	//获取feed列表
	msgs, err := c.muxi.GetToBePublicOfficialMSG(ctx, true)
	if err != nil {
		c.l.Warn(saramax.LogKeyConsumeRetry+" 获取木犀消息失败!", logger.Error(err))
		return
	}
	if len(msgs) == 0 {
		return
	}

	for _, msg := range msgs {
		event := domain.FeedEvent{
			DedupeKey:    "muxi:" + msg.Id,
			Type:         strings.ToLower(feedv1.FeedEventType_MUXI.String()),
			Title:        msg.Title,
			Content:      msg.Content,
			ExtendFields: msg.ExtendFields,
		}
		// 管理端录入的内容可能永远无法入库（校验失败）且重试无法修复，
		// 一直留在队列里会让后续所有木犀消息都发不出去，只能记录后删除该计划。
		check := event
		check.StudentId = "broadcast"
		if err := domain.ValidateFeedEventForStorage(check); err != nil {
			c.l.Error(saramax.LogKeyMessageDropped+" 木犀消息无法入库，删除该计划",
				logger.String("msg_id", msg.Id), logger.Error(err))
			if stopErr := c.muxi.StopMuxiOfficialMSG(ctx, msg.Id); stopErr != nil {
				c.l.Warn("删除无法发布的木犀消息失败", logger.String("msg_id", msg.Id), logger.Error(stopErr))
				return
			}
			continue
		}

		//发布消息给全体成员
		_, err = c.feed.PublicFeedEvent(ctx, true, event)
		if err != nil {
			c.l.Warn(saramax.LogKeySendFailed+" 消息推送失败!", logger.String("msg_id", msg.Id), logger.Error(err))
			return
		}
		// 全部收件人发布成功后再移除计划，失败或重启后仍能用原 key 补发。
		if err := c.muxi.StopMuxiOfficialMSG(ctx, msg.Id); err != nil {
			c.l.Warn("确认木犀消息发布完成失败", logger.Error(err))
			return
		}
	}

	return
}
