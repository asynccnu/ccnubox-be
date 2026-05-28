package ioc

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-feed/conf"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/otel"
)

func InitOTel(cfg *conf.ServerConf) func(ctx context.Context) error {
	return otel.InitOTel(cfg.Otel)
}
