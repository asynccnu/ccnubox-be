package health

import (
	"log"

	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/web"
	"github.com/gin-gonic/gin"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type HealthHandler struct {
	clients map[string]healthpb.HealthClient
}

func NewHealthHandler(clients map[string]healthpb.HealthClient) *HealthHandler {
	return &HealthHandler{
		clients: clients,
	}
}

func (h *HealthHandler) RegisterRoutes(s *gin.RouterGroup, basicAuthMiddleware gin.HandlerFunc) {
	s.GET("/health/live", basicAuthMiddleware, ginx.Wrap(h.HealthCheck))
	s.GET("/health/ready", basicAuthMiddleware, ginx.Wrap(h.ReadyCheck))
}

// HealthCheck 健康存活检查
// @Summary 健康存活检查
// @Description 返回服务存活状态，使用 BasicAuth 进行验证
// @Tags health
// @Produce json
// @Success 200 {object} web.Response "ok"
// @Router /health/live [get]
// @Security BasicAuth
func (h *HealthHandler) HealthCheck(c *gin.Context) (web.Response, error) {
	return web.Response{
		Code: 200,
		Msg:  "ok",
	}, nil
}

// ReadyCheck 依赖服务就绪检查
// @Summary 依赖服务就绪检查
// @Description 检查各依赖服务的健康状态，使用 BasicAuth 进行验证
// @Tags health
// @Produce json
// @Success 200 {object} web.Response{data=map[string]string} "ok"
// @Router /health/ready [get]
// @Security BasicAuth
func (h *HealthHandler) ReadyCheck(c *gin.Context) (web.Response, error) {
	var res = make(map[string]string)
	for n, client := range h.clients {
		resp, err := client.Check(c, &healthpb.HealthCheckRequest{})
		if err != nil {
			res[n] = err.Error()
			log.Printf("服务 %s 健康检查失败: %v", n, err)
		}
		res[n] = resp.Status.String()
	}
	return web.Response{
		Code: 200,
		Msg:  "ok",
		Data: res,
	}, nil
}
