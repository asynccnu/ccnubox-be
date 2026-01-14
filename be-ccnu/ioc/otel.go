package ioc

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-ccnu/conf"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/otel"
)

// InitOTel 初始化
func InitOTel(cfg *conf.ServerConf) func(ctx context.Context) error {
	return otel.InitOTel(cfg.Otel)
}
