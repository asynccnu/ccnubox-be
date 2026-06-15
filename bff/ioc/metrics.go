package ioc

import (
	"context"

	"github.com/asynccnu/ccnubox-be/bff/conf"
	"github.com/asynccnu/ccnubox-be/bff/cron"
	"github.com/asynccnu/ccnubox-be/common/pkg/cronx"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/asynccnu/ccnubox-be/common/pkg/metricsx"
	"github.com/go-redsync/redsync/v4"
	"github.com/redis/go-redis/v9"
)

const dauCronSpec = "5 0 * * *" // 每天 00:05:00 触发

func InitMetrics(cfg *conf.ServerConf) *metricsx.Metrics {
	return metricsx.New("ccnubox")
}

// InitDAURefresher 注册 DAURefresher 到 cronx:
//  1. 启动时 Bootstrap: 从 Redis 恢复最近一次成功值, 避免 gauge 起始为 0。
//  2. 注册每日 00:05 任务: PFMERGE 昨天 → gauge.Set + 写 Redis 兜底。
//
// Gauge 自身已在 metricsx.NewWithRegisterer 中注册到 Prometheus 默认 registry,
// BFF 的 /metrics 端点会通过 promhttp 自动暴露 ccnubox_dau 一条 series。
//
// 返回 DAURefresher 引用是为了让 App 持有 (注入 wire 依赖图), 未来可挂入
// 优雅关闭流程; 当前 Start() 尚未使用, 仅占位。
func InitDAURefresher(
	m *metricsx.Metrics,
	redisClient redis.Cmdable,
	rs *redsync.Redsync,
	cronMgr *cronx.Manager,
	l logger.Logger,
) *cron.DAURefresher {
	refresher := cron.NewDAURefresher(redisClient, m.User.DAU, rs, l)

	// 启动兜底: 读 Redis 上一次成功值。
	refresher.Bootstrap(context.Background())

	// 注册每日 00:05 cron 任务。
	if err := cronMgr.AddTask("dau_refresh", dauCronSpec, func(ctx context.Context, log logger.Logger) {
		refresher.Refresh(ctx)
	}); err != nil {
		// 注册失败只 warn: 启动期不应阻塞进程, 但日志必须显式记录便于排查。
		l.Warn("dau refresh: register cron task failed", logger.Error(err))
	}

	return refresher
}
