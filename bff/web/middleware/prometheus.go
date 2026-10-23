package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/asynccnu/ccnubox-be/bff/cron"
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
			// Record authenticated users in the current 15-minute HLL bucket.
			uc, _ := ginx.GetClaims[ijwt.UserClaims](ctx)
			studentID := uc.StudentId
			if studentID != "" {
				go func(studentID string) {
					// Only the approximate metric is best-effort. Exact activity
					// intents have already been persisted by LoginMiddleware.
					ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
					defer cancel()

					_ = m.recordActiveUser(ctx, studentID, start)
				}(studentID)
			}

			status := ctx.Writer.Status()
			m.metrics.HTTP.ActiveConnections.WithLabelValues(path).Dec()
			m.metrics.HTTP.RequestsTotal.WithLabelValues(ctx.Request.Method, path, http.StatusText(status)).Inc()
			m.metrics.HTTP.Duration.WithLabelValues(path, http.StatusText(status)).Observe(time.Since(start).Seconds())
		}()

		ctx.Next()
	}
}

func (m *PrometheusMiddleware) recordActiveUser(ctx context.Context, studentID string, now time.Time) error {
	if studentID == "" {
		return nil
	}

	bucketKey := cron.ActiveUsersBucketKey(now)
	if err := m.redisClient.PFAdd(ctx, bucketKey, studentID).Err(); err != nil {
		return err
	}
	if err := m.redisClient.Expire(ctx, bucketKey, cron.ActiveUsersBucketTTL()).Err(); err != nil {
		return err
	}
	return nil
}
