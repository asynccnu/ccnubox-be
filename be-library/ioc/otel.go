package ioc

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-library/conf"
	bgrpc "github.com/asynccnu/ccnubox-be/common/bizpkg/grpc"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/otel"
)

// OTelShutdownFunc 独立命名 OTel 关闭函数类型，避免 wire 对 func(context.Context) error 的绑定产生歧义。
type OTelShutdownFunc func(context.Context) error

// InitOTel 初始化 OpenTelemetry Provider，为后台提醒任务与 gRPC 调用提供链路追踪。
func InitOTel(infraCfg *conf.InfraConf) OTelShutdownFunc {
	return otel.InitOTelFromInfra(infraCfg.InfraConf, bgrpc.LIBRARY)
}
