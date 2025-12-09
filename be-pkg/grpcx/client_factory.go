package grpcx

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	googlegrpc "google.golang.org/grpc"
)

func Dial(ctx context.Context, r registry.Discovery, endpoint string) *googlegrpc.ClientConn {
	cc, err := grpc.DialInsecure(ctx,
		grpc.WithEndpoint(endpoint),
		grpc.WithDiscovery(r),
		grpc.WithTimeout(30*time.Second),
		grpc.WithMiddleware(
			tracing.Client(),
		),
	)

	if err != nil {
		panic(err)
	}

	return cc
}
