package middleware

import (
	"net/http"
	"time"

	"github.com/asynccnu/ccnubox-be/bff/pkg/prometheusx"
	"github.com/gin-gonic/gin"
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
		// path := ctx.Request.URL.Path
		path := ctx.FullPath()
		if path == "" {
			path = "not found"
		}
		m.prometheus.ActiveConnections.WithLabelValues(path).Inc()
		defer func() {
			// 由于学号数据量太过高基数且太过离散，这里的数据从个体请求数的采集改为总体请求数

			// TODO 这里没有想到更加简单便捷的方案去判断是否需要记录学号,所以全都记录了
			// var studentId = "no studentId"
			// uc, _ := ginx.GetClaims[ijwt.UserClaims](ctx)
			// if uc.StudentId != "" {
			// 	studentId = uc.StudentId
			// }

			// 记录响应信息
			m.prometheus.ActiveConnections.WithLabelValues(path).Dec()
			status := ctx.Writer.Status()
			m.prometheus.RouterCounter.WithLabelValues(ctx.Request.Method, path, http.StatusText(status)).Inc()
			m.prometheus.DurationTime.WithLabelValues(path, http.StatusText(status)).Observe(time.Since(start).Seconds())

		}()

		ctx.Next() // 执行后续逻辑

	}
}
