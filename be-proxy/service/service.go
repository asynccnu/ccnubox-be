package service

import (
	"context"
	"github.com/google/wire"
)

type ProxyService interface {
	GetProxyAddr(ctx context.Context) (string, error)
}

var Provider = wire.NewSet(
	NewProxyService,
)
