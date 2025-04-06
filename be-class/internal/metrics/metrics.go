package metrics

import (
	"context"
	"fmt"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/prometheus/client_golang/prometheus"
	"time"
)

var (
	// Counter 定义 CounterVec 指标采集器
	Counter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_request_total",
			Help: "The total number of HTTP requests",
		},
		[]string{"uri"},
	)
	// Summary 定义Summary指标采集器
	Summary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "http_request_delay",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"uri"},
	)
)

// QPSMiddleware 记录QPS
func QPSMiddleware() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			if tr, ok := transport.FromServerContext(ctx); ok {
				uri := fmt.Sprintf("%s://%s", tr.Kind(), tr.Operation())
				Counter.WithLabelValues(uri).Inc()
			}
			return handler(ctx, req)
		}
	}
}

// DelayMiddleware 记录请求时长
func DelayMiddleware() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			if tr, ok := transport.FromServerContext(ctx); ok {
				uri := fmt.Sprintf("%s://%s", tr.Kind(), tr.Operation())
				startTime := time.Now()
				defer func() {
					latency := time.Now().Sub(startTime)
					Summary.WithLabelValues(uri).
						Observe(float64(latency.Milliseconds()))
				}()
			}
			return handler(ctx, req)
		}
	}
}
