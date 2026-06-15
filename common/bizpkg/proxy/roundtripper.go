package proxy

import (
	"net/http"
	"time"
)

func (h *HttpTransport) Use(options ...RoundTripperOption) {
	if len(options) == 0 {
		return
	}
	for _, option := range options {
		option(h)
	}
}

type HttpTransport struct {
	*http.Transport
}
type RoundTripperOption func(transport *HttpTransport)

func WithKeepAliveDisabled(disabled bool) RoundTripperOption {
	return func(transport *HttpTransport) {
		transport.DisableKeepAlives = disabled
	}
}

func WithResponseHeaderTimeout(responseHeaderTimeout time.Duration) RoundTripperOption {
	return func(transport *HttpTransport) {
		transport.ResponseHeaderTimeout = responseHeaderTimeout
	}
}

func WithMaxIdleConns(maxIdleConns int) RoundTripperOption {
	return func(transport *HttpTransport) {
		transport.MaxIdleConns = maxIdleConns
	}
}

func WithIdleConnTimeout(idleConnTimeout time.Duration) RoundTripperOption {
	return func(transport *HttpTransport) {
		transport.IdleConnTimeout = idleConnTimeout
	}
}

func WithMaxConnsPerHost(maxConnsPerHost int) RoundTripperOption {
	return func(transport *HttpTransport) {
		transport.MaxConnsPerHost = maxConnsPerHost
	}
}

func WithTLSHandshakeTimeout(tlsHandshakeTimeout time.Duration) RoundTripperOption {
	return func(transport *HttpTransport) {
		transport.TLSHandshakeTimeout = tlsHandshakeTimeout
	}
}
