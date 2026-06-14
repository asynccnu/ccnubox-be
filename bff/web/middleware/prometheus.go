package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	"github.com/asynccnu/ccnubox-be/common/pkg/metricsx"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type PrometheusMiddleware struct {
	metrics     *metricsx.Metrics
	redisClient redis.Cmdable
}

func NewPrometheusMiddleware(
	metrics *metricsx.Metrics,
	redisClient redis.Cmdable,
) *PrometheusMiddleware {
	return &PrometheusMiddleware{
		metrics:     metrics,
		redisClient: redisClient,
	}
}

func (m *PrometheusMiddleware) MiddlewareFunc() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()

		path := ctx.FullPath()
		if path == "" {
			path = "not found"
		}
		m.metrics.HTTP.ActiveConnections.WithLabelValues(path).Inc()

		defer func() {
			// 向 redis 存入学号数据
			uc, _ := ginx.GetClaims[ijwt.UserClaims](ctx)
			StudentId := uc.StudentId
			if StudentId != "" {
				// 存入redis进行聚合数据处理日活数据
				// 这里将每个键分为15min的桶，实现精度较高的滑动窗口
				go func(studentId string) {
					// DAU 聚合是 best-effort, 失败会通过下面的 InstrumentedRedis 记录到
					// ccnubox_redis_errors_total{operation="PFADD"|"EXPIRE"} 指标里。
					// 告警建议: rate(ccnubox_redis_errors_total{operation=~"PFADD|EXPIRE"}[5m]) > 0
					now := time.Now()
					bucketTime := now.Truncate(15 * time.Minute)
					key := "dau:" + bucketTime.Format("2006-01-02-15-04")

					ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
					defer cancel()

					m.redisClient.PFAdd(ctx, key, studentId)
					m.redisClient.Expire(ctx, key, 26*time.Hour)
				}(StudentId)
			}

			// 记录响应信息
			status := ctx.Writer.Status()
			m.metrics.HTTP.ActiveConnections.WithLabelValues(path).Dec()
			m.metrics.HTTP.RequestsTotal.WithLabelValues(ctx.Request.Method, path, http.StatusText(status)).Inc()
			m.metrics.HTTP.Duration.WithLabelValues(path, http.StatusText(status)).Observe(time.Since(start).Seconds())
		}()

		ctx.Next() // 执行后续逻辑
	}
}
