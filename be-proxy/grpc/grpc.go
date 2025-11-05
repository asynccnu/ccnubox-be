package grpc

import "github.com/google/wire"

var Provider = wire.NewSet(
	NewProxyServiceServer,
)
