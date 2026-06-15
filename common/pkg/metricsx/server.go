package metricsx

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// metricsPortOffset metrics HTTP 端口相对于 gRPC 端口的偏移量,
// 约定 metrics 端口 = grpc 端口 + 1000, 避免 yaml 单独配置。
const metricsPortOffset = 1000

type Server struct {
	addr string

	mu     sync.Mutex
	server *http.Server
}

// NewServer 直接基于 addr 构造, 主要给测试或特殊场景使用。
// 业务侧请用 NewServerFromGRPCAddr 以保持端口派生约定的统一。
func NewServer(addr string) *Server {
	return &Server{addr: addr}
}

// NewServerFromGRPCAddr 基于 gRPC 监听地址派生 metrics 监听地址,
// 端口取 gRPC 端口 + metricsPortOffset, host 沿用 gRPC 配置。
// grpcAddr 解析失败时 fallback 到 ":29090", 不阻断启动。
func NewServerFromGRPCAddr(grpcAddr string) *Server {
	return NewServer(DeriveAddr(grpcAddr))
}

// DeriveAddr 从 gRPC 监听 addr 派生出 metrics 监听 addr。
// 期望输入形如 ":29084" 或 "0.0.0.0:29084"; 解析失败返回 ":29090"。
func DeriveAddr(grpcAddr string) string {
	host, port, err := net.SplitHostPort(grpcAddr)
	if err != nil {
		return ":29090"
	}
	p, err := strconv.Atoi(port)
	if err != nil {
		return ":29090"
	}
	return net.JoinHostPort(host, strconv.Itoa(p+metricsPortOffset))
}

// Addr 返回当前监听地址, 供日志/调试使用。
func (s *Server) Addr() string {
	if s == nil {
		return ""
	}
	return s.addr
}

func (s *Server) Serve() error {
	if s == nil || s.addr == "" {
		return nil
	}

	handler := promhttp.Handler()
	mux := http.NewServeMux()
	// /metrics 兼容社区惯例与默认 Prometheus 抓取路径;
	// /api/v1/metrics 兼容仓库内既有的 prometheus.yml 与 bff 路由风格。
	mux.Handle("/metrics", handler)
	mux.Handle("/api/v1/metrics", handler)

	// 注意: gRPC 服务的 metrics 端点目前无鉴权, 依赖"集群内网可信任"的部署模型。
	// 如果未来需要暴露到公网/办公网, 应在 mux 外面套一层 basic auth 或 IP allowlist 中间件。
	server := &http.Server{
		Addr:    s.addr,
		Handler: mux,
		// ReadHeader/Read/Write/Idle 超时是为了避免恶意/慢客户端拖死 listener。
		// metrics 端点只读、payload 小, 给到较紧的超时即可。
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	s.mu.Lock()
	s.server = server
	s.mu.Unlock()

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) Close() error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	server := s.server
	s.mu.Unlock()
	if server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return server.Shutdown(ctx)
}
