package server

import (
	"time"

	"github.com/asynccnu/ccnubox-be/be-classlist/internal/conf"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/metrics"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/service"
	v1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/classlist/v1"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/ratelimit"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(c *conf.Server, greeter *service.ClassListService, logger log.Logger) *grpc.Server {
	var opts = []grpc.ServerOption{
		grpc.Middleware(
			recovery.Recovery(),
			metrics.QPSMiddleware(),
			metrics.DelayMiddleware(),
			tracing.Server(),
			logging.Server(logger),
			//validate.Validator(),
			ratelimit.Server(),
		),
	}
	if c.Grpc.Network != "" {
		opts = append(opts, grpc.Network(c.Grpc.Network))
	}
	if c.Grpc.Addr != "" {
		opts = append(opts, grpc.Address(c.Grpc.Addr))
	}
	if c.Grpc.Timeout > 0 {
		opts = append(opts, grpc.Timeout(time.Duration(c.Grpc.Timeout)*time.Second))
	}
	srv := grpc.NewServer(opts...)
	v1.RegisterClasserServer(srv, greeter)
	return srv
}
