package ioc

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-counter/conf"
	bgrpc "github.com/asynccnu/ccnubox-be/common/bizpkg/grpc"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/otel"
)

func InitOTel(infraCfg *conf.InfraConf) func(ctx context.Context) error {
	return otel.InitOTelFromInfra(infraCfg.InfraConf, bgrpc.COUNTER)
}
