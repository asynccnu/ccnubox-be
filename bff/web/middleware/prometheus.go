package middleware

import (
	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/pkg/prometheusx"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

type PrometheusMiddleware struct {
	prometheus *prometheusx.PrometheusCounter
}

func NewPrometheusMiddleware(
	prometheus *prometheusx.PrometheusCounter,
) *PrometheusMiddleware {
	return &PrometheusMiddleware{
		prometheus: prometheus,
	}
}

func (m *PrometheusMiddleware) MiddlewareFunc() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()
		path := ctx.Request.URL.Path
		defer func() {
			m.prometheus.ActiveConnections.WithLabelValues(path).Inc()

			// TODO 这里没有想到更加简单便捷的方案去判断是否需要记录学号,所以全都记录了
			var studentId = "no studentId"
			uc, _ := ginx.GetClaims[ijwt.UserClaims](ctx)
			if uc.StudentId != "" {
				studentId = uc.StudentId
			}

			// 记录响应信息
			m.prometheus.ActiveConnections.WithLabelValues(path).Dec()
			status := ctx.Writer.Status()
			m.prometheus.RouterCounter.WithLabelValues(ctx.Request.Method, path, http.StatusText(status), studentId).Inc()
			m.prometheus.DurationTime.WithLabelValues(path, http.StatusText(status)).Observe(time.Since(start).Seconds())

		}()

		ctx.Next() // 执行后续逻辑

	}
}
