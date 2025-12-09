package otelx

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
)

type config struct {
	sampler  trace.Sampler
	resource *resource.Resource
	endpoint string
}

type Option func(*config)

// 选择 Sampler
func WithSampler(s trace.Sampler) Option {
	return func(c *config) {
		c.sampler = s
	}
}

// 选择 Resource
func WithResource(r *resource.Resource) Option {
	return func(c *config) {
		c.resource = r
	}
}

// 选择数据暴露地址
func WithEndpoint(addr string) Option {
	return func(c *config) {
		c.endpoint = addr
	}
}

// setupOTel 初始化 OpenTel
func SetupOTel(ctx context.Context, opts ...Option) (func(context.Context) error, error) {
	defaultResource := resource.Default()
	cfg := config{
		sampler:  trace.AlwaysSample(),
		resource: defaultResource,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	var shutdownFuncs []func(context.Context) error
	var err error

	// shutdown 会调用所有注册的清理函数
	// 所有返回的错误都会合并到一起
	// 每个注册的清理函数仅会被调用一次
	shutdown := func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	// handleErr 用户调用 shutdown 并合并返回的错误
	// 包裹错误 + 优雅退出
	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	// 设置上下文传播器（用于跨服务传递追踪信息）
	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	// 初始化 trace 提供者
	tracerProvider, err := newTracerProvider(ctx, cfg)
	if err != nil {
		handleErr(err)
		return shutdown, err
	}
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	return shutdown, nil
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

// newTracerProvider 接受 config
func newTracerProvider(ctx context.Context, cfg config) (*trace.TracerProvider, error) {
	traceExporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(cfg.endpoint),
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithTimeout(5*time.Second),
	)
	if err != nil {
		return nil, err
	}

	tracerProvider := trace.NewTracerProvider(
		trace.WithSampler(cfg.sampler),
		trace.WithResource(cfg.resource),
		trace.WithBatcher(traceExporter),
	)

	return tracerProvider, nil
}
