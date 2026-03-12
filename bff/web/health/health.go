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

func (h *HealthHandler) HealthCheck(c *gin.Context) (web.Response, error) {
	return web.Response{
		Code: 200,
		Msg:  "ok",
	}, nil
}

func (h *HealthHandler) ReadyCheck(c *gin.Context) (web.Response, error) {
	var res = make(map[string]string)
	for n, client := range h.clients {
		resp, err := client.Check(c, &healthpb.HealthCheckRequest{})
		if err != nil {
			res[n] = err.Error()
			log.Printf("服务 %s 健康检查失败: %v", n, err)
		}
		res[n] = resp.String()
	}
	return web.Response{
		Code: 200,
		Msg:  "ok",
		Data: res,
	}, nil
}
