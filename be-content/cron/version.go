package cron

import (
	"context"
	"time"

	"github.com/asynccnu/ccnubox-be/be-content/service"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

type UpdateVersionController struct {
	svc      service.VersionService
	l        logger.Logger
	interval time.Duration
}

func NewUpdateVersionController(svc service.VersionService, l logger.Logger) *UpdateVersionController {
	return &UpdateVersionController{
		svc:      svc,
		l:        l,
		interval: 10 * time.Minute,
	}
}

func (r *UpdateVersionController) StartCronTask() {
	go func() {
		ticker := time.NewTicker(r.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				err := r.svc.Refresh(context.Background())
				if err != nil {
					r.l.Error("版本刷新失败", logger.Error(err))
				}
			}
		}
	}()
}
