package ioc

import (
	"context"
	"fmt"

	"github.com/asynccnu/ccnubox-be/bff/conf"
	"github.com/asynccnu/ccnubox-be/common/pkg/otelx"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

// InitOTel 初始化
func InitOTel(cfg *conf.TransConf) func(ctx context.Context) error {
	// 构造 Resource
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.Otel.ServiceName),
			semconv.ServiceVersion(cfg.Otel.ServiceVersion),
		),
	)
	if err != nil {
		panic(fmt.Sprintf("otel 创建 resource 失败：%v", err))
	}

	// 初始化 OTel
	shutdown, err := otelx.SetupOTel(
		context.Background(),
		otelx.WithResource(res),
		otelx.WithEndpoint(cfg.Otel.Endpoint),
	)
	if err != nil {
		panic(fmt.Sprintf("otel 初始化失败: %v", err))
	}

	return shutdown
}
