package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/asynccnu/ccnubox-be/be-pkg/ginx"
	"github.com/asynccnu/ccnubox-be/be-pkg/prometheusx"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type PrometheusMiddleware struct {
	prometheus  *prometheusx.PrometheusCounter
	redisClient redis.Cmdable
}

func NewPrometheusMiddleware(
	prometheus *prometheusx.PrometheusCounter,
	redisClient redis.Cmdable,
) *PrometheusMiddleware {
	return &PrometheusMiddleware{
		prometheus:  prometheus,
		redisClient: redisClient,
	}
}

func (m *PrometheusMiddleware) MiddlewareFunc() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()

		// path := ctx.Request.URL.Path
		path := ctx.FullPath()
		if path == "" {
			path = "not found"
		}
		m.prometheus.ActiveConnections.WithLabelValues(path).Inc()

		defer func() {
			// 向 redis 存入学号数据
			uc, _ := ginx.GetClaims[ijwt.UserClaims](ctx)
			StudentId := uc.StudentId
			if StudentId != "" {
				// 存入redis进行聚合数据处理日活数据
				// 这里将每个键分为15min的桶，实现精度较高的滑动窗口
				go func(studentId string) {
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
			m.prometheus.ActiveConnections.WithLabelValues(path).Dec()
			m.prometheus.RouterCounter.WithLabelValues(ctx.Request.Method, path, http.StatusText(status)).Inc()
			m.prometheus.DurationTime.WithLabelValues(path, http.StatusText(status)).Observe(time.Since(start).Seconds())
		}()

		ctx.Next() // 执行后续逻辑
	}
}
