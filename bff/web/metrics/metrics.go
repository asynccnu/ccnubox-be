package metrics

import (
	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/web"
	"github.com/gin-gonic/gin"
)

type MetricsHandler struct{}

func NewMetricsHandler() *MetricsHandler {
	return &MetricsHandler{}
}

func (h *MetricsHandler) RegisterRoutes(s *gin.RouterGroup, authMiddleware gin.HandlerFunc) {

	//用于给前端自动打点的路由,暂时不做额外参数处理
	s.POST("/metrics/:eventName", authMiddleware, ginx.Wrap(h.Metrics))
}

// Metrics 用于打点的路由
// @Summary 用于打点的路由
// @Description 用于打点的路由,如果是不经过后端的服务但是需要打点的话,可以使用这个路由自动记录(例如:/metrics/kstack)表示跳转访问课栈,使用这一路由必须携带Auth请求头
// @Tags 打点
// @Success 200 {object} web.Response{} "成功"
// @Router /metrics/:eventName [post]
func (h *MetricsHandler) Metrics(ctx *gin.Context) (web.Response, error) {
	// 获取路由中的参数 eventName
	eventName := ctx.Param("eventName")

	// 将 eventName 作为 message 的一部分返回
	return web.Response{
		Msg: "事件: " + eventName, // 拼接 message
	}, nil
}
