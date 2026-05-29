package otel

import (
	"context"
	"fmt"

	"github.com/asynccnu/ccnubox-be/common/bizpkg/conf"
	bgrpc "github.com/asynccnu/ccnubox-be/common/bizpkg/grpc"
	"github.com/asynccnu/ccnubox-be/common/pkg/otelx"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

func InitOTelFromInfra(infraCfg *conf.InfraConf, serviceKey string) func(ctx context.Context) error {
	cfg := &conf.OtelConf{
		ServiceName: bgrpc.GetNamePrefix(infraCfg.Env, (*infraCfg.Grpc)[serviceKey].Name),
		Endpoint:    infraCfg.Otel.Endpoint,
	}
	return InitOTel(cfg)
}

// InitOTel 初始化
func InitOTel(cfg *conf.OtelConf) func(ctx context.Context) error {
	// 构造 Resource
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
		),
	)
	if err != nil {
		panic(fmt.Sprintf("otel 创建 resource 失败：%v", err))
	}

	// 初始化 OTel
	shutdown, err := otelx.SetupOTel(
		context.Background(),
		otelx.WithResource(res),
		otelx.WithEndpoint(cfg.Endpoint),
	)
	if err != nil {
		panic(fmt.Sprintf("otel 初始化失败: %v", err))
	}

	return shutdown
}
