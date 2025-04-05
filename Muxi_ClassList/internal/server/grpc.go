package server

import (
	"github.com/asynccnu/ccnubox-be/Muxi_ClassList/internal/conf"
	"github.com/asynccnu/ccnubox-be/Muxi_ClassList/internal/metrics"
	"github.com/asynccnu/ccnubox-be/Muxi_ClassList/internal/service"
	v1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/classlist/v1"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/ratelimit"
	"github.com/go-kratos/kratos/v2/middleware/validate"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(c *conf.Server, greeter *service.ClasserService, logger log.Logger) *grpc.Server {
	var opts = []grpc.ServerOption{
		grpc.Middleware(
			recovery.Recovery(),
			metrics.QPSMiddleware(),
			metrics.DelayMiddleware(),
			logging.Server(logger),
			validate.Validator(),
			ratelimit.Server(),
		),
	}
	if c.Grpc.Network != "" {
		opts = append(opts, grpc.Network(c.Grpc.Network))
	}
	if c.Grpc.Addr != "" {
		opts = append(opts, grpc.Address(c.Grpc.Addr))
	}
	if c.Grpc.Timeout != nil {
		opts = append(opts, grpc.Timeout(c.Grpc.Timeout.AsDuration()))
	}
	srv := grpc.NewServer(opts...)
	v1.RegisterClasserServer(srv, greeter)
	return srv
}
