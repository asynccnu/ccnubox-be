package ioc

import (
	"context"
	"fmt"

	"github.com/asynccnu/ccnubox-be/bff/pkg/otelx"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

type OtelConfig struct {
	ServiceName    string `yaml:"serviceName"`
	ServiceVersion string `yaml:"serviceVersion"`
	Endpoint       string `yaml:"endpoint"`
}

// 提供给中间件优雅关闭函数
// InitOTel 初始化
func InitOTel() func(ctx context.Context) error {
	var cfg OtelConfig

	// 读取配置
	err := viper.UnmarshalKey("otel", &cfg)
	if err != nil {
		panic(fmt.Sprintf("otel 读取配置失败：%v", err))
	}

	// 构造 Resource
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
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
