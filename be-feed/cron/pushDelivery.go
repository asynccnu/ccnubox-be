package cron

import (
	"context"
	"sync"
	"time"

	"github.com/asynccnu/ccnubox-be/be-feed/service"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

type PushDeliveryController struct {
	delivery service.PushDeliveryService
	log      logger.Logger
	ctx      context.Context
	cancel   context.CancelFunc
	stopOnce sync.Once
	wg       sync.WaitGroup
}

func NewPushDeliveryController(delivery service.PushDeliveryService, log logger.Logger) *PushDeliveryController {
	ctx, cancel := context.WithCancel(context.Background())
	return &PushDeliveryController{delivery: delivery, log: log, ctx: ctx, cancel: cancel}
}

func (c *PushDeliveryController) StartCronTask() error {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	err := c.delivery.RecoverSending(ctx)
	cancel()
	if err != nil {
		return err
	}
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
				err := c.delivery.DispatchDue(ctx)
				cancel()
				if err != nil {
					c.log.Error("dispatch push deliveries failed", logger.Error(err))
				}
			case <-c.ctx.Done():
				return
			}
		}
	}()
	return nil
}

func (c *PushDeliveryController) StopCronTask() {
	c.stopOnce.Do(c.cancel)
	c.wg.Wait()
}
