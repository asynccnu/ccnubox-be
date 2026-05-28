package ioc

import (
	"context"

	"github.com/asynccnu/ccnubox-be/bff/conf"
	baseconf "github.com/asynccnu/ccnubox-be/common/bizpkg/conf"
	bgrpc "github.com/asynccnu/ccnubox-be/common/bizpkg/grpc"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/otel"
)

// InitOTel 初始化
func InitOTel(infraCfg *conf.InfraConf) func(ctx context.Context) error {
	serviceName := infraCfg.Otel.ServiceName
	if serviceName == "" {
		serviceName = "bff"
	}

	cfg := &baseconf.OtelConf{
		ServiceName: bgrpc.GetNamePrefix(infraCfg.Env, serviceName),
		Endpoint:    infraCfg.Otel.Endpoint,
	}
	return otel.InitOTel(cfg)
}
