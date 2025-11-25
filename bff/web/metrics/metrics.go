package metrics

import (
	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/pkg/logger"
	"github.com/asynccnu/ccnubox-be/bff/web"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type MetricsHandler struct {
	l logger.Logger
}

func NewMetricsHandler(l logger.Logger) *MetricsHandler {
	return &MetricsHandler{
		l: l,
	}
}

func (h *MetricsHandler) RegisterRoutes(s *gin.RouterGroup, basicAuthMiddleware gin.HandlerFunc, authMiddleware gin.HandlerFunc) {

	s.GET("/metrics", basicAuthMiddleware, h.MetricsExporter)
	//用于给前端自动打点的路由,暂时不做额外参数处理
	s.POST("/metrics/:type/:name", authMiddleware, ginx.WrapClaimsAndReq(h.Metrics))
}

// MetricsExporter 导出 Prometheus 监控指标
// @Summary 导出 Prometheus 监控指标
// @Description 暴露标准的 Prometheus 监控数据，供 Prometheus 定时拉取。
// @Tags metrics
// @Produce text/plain
// @Success 200 {string} string "Prometheus Exporter Text Data"
// @Router /metrics [get]
// @Security BasicAuth
func (h *MetricsHandler) MetricsExporter(c *gin.Context) {
	promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	c.Abort()
}

// Metrics 用于打点的路由
// @Summary 用于打点的路由
// @Description 用于打点的路由,如果是不经过后端的服务但是需要打点的话,可以使用这个路由自动记录(例如:/metrics/banner/xxx)表示跳转banner的xxx页面,使用这一路由必须携带Auth请求头
// @Tags metrics
// @Param data body MetricsReq true "打点附带的信息,将会计入日志"
// @Success 200 {object} web.Response{} "成功"
// @Router /metrics/:type/:name [post]
func (h *MetricsHandler) Metrics(ctx *gin.Context, req MetricsReq, uc ijwt.UserClaims) (web.Response, error) {
	// 获取路由中的参数 t
	t := ctx.Param("type")
	name := ctx.Param("name")

	fields := []logger.Field{
		logger.String("path", "/api/v1/metrics/"+t+"/"+name),
		logger.String("msg", req.Msg),
		logger.String("user:", uc.StudentId),
	}

	switch req.Level {
	case "warn":
		h.l.Warn("metrics", fields...)
	case "info":
		h.l.Info("metrics", fields...)

	case "error":
		h.l.Error("metrics", fields...)

	case "debug":
		h.l.Debug("metrics", fields...)

	default:
		h.l.Warn("metrics", fields...)

	}

	// 将 t 作为 message 的一部分返回
	return web.Response{
		Msg: "事件: " + t + "/" + name + "打点成功!", // 拼接 message
	}, nil
}
