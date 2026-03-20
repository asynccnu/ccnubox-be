package ioc

import (
	"github.com/asynccnu/ccnubox-be/bff/conf"
	counterv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/counter/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/asynccnu/ccnubox-be/common/pkg/tieredx"
	"github.com/go-redsync/redsync/v4"
)

func InitScheduler(cfg *conf.ServerConf, handler tieredx.RefreshHandler, counter counterv1.CounterServiceClient, l logger.Logger, rs *redsync.Redsync) *tieredx.TieredScheduler {
	tieredConf := tieredx.TieredConfig{
		Low:    cfg.Tiered.Low,
		Middle: cfg.Tiered.Middle,
		High:   cfg.Tiered.High,
	}
	return tieredx.NewTieredScheduler(tieredConf, handler, counter, l, tieredx.WithRedsync(rs))
}
