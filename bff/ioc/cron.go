package ioc

import (
	"context"

	"github.com/asynccnu/ccnubox-be/bff/cron"
	"github.com/asynccnu/ccnubox-be/common/pkg/cronx"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/asynccnu/ccnubox-be/common/pkg/metricsx"
	"github.com/go-redsync/redsync/v4"
	"github.com/redis/go-redis/v9"
)

const dauCronSpec = "5 0 * * *" // every day 00:05

func InitCronxManager(
	l logger.Logger,
	m *metricsx.Metrics,
	redisClient redis.Cmdable,
	rs *redsync.Redsync,
) *cronx.Manager {
	manager := cronx.NewManager(l)
	registerDAURefreshTask(manager, m, redisClient, rs, l)
	return manager
}

func registerDAURefreshTask(
	cronMgr *cronx.Manager,
	m *metricsx.Metrics,
	redisClient redis.Cmdable,
	rs *redsync.Redsync,
	l logger.Logger,
) {
	refresher := cron.NewDAURefresher(redisClient, m.User.DAU, rs, l)
	refresher.Bootstrap(context.Background())

	if err := cronMgr.AddTask("dau_refresh", dauCronSpec, func(ctx context.Context, log logger.Logger) {
		refresher.Refresh(ctx)
	}); err != nil {
		l.Warn("dau refresh: register cron task failed", logger.Error(err))
	}
}
