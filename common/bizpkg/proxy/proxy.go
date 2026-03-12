package proxy

import (
	"context"
)

type Getter interface {
	GetProxyAddr(ctx context.Context, cnt int) []string
}

type Client interface {
	NewProxyClient(ops ...Option) *HttpClient
}

type Transporter interface {
	NewProxyTransport(ops ...RoundTripperOption) *HttpTransport
}
