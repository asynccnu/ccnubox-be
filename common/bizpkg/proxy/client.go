package proxy

import (
	"context"
	"net/http"
	"time"
)

type HttpClient struct {
	*http.Client
}

func (c *HttpClient) Use(options ...Option) {
	for _, option := range options {
		option(c)
	}
}

var (
	RedirectPolicyDeny = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	RedirectPolicyAllow = func(req *http.Request, via []*http.Request) error {
		return nil
	}
	RedirectPolicyDefault = http.DefaultClient.CheckRedirect
)

type Option func(*HttpClient)

func WithTransport(tr *HttpTransport) Option {
	return func(client *HttpClient) {
		client.Transport = tr
	}
}

func WithProxyTransport(isBackup bool, options ...RoundTripperOption) Option {
	return func(client *HttpClient) {
		proxyAddrs := globalProxy.GetProxyAddr(context.Background(), 2)
		var proxyAddr string
		if isBackup {
			proxyAddr = proxyAddrs[1]
		} else {
			proxyAddr = proxyAddrs[0]
		}
		tr := globalProxy.NewProxyTransport(
			WithProxy(proxyAddr),
		)

		tr.Use(options...)
		client.Transport = tr
	}
}

func WithRedirectPolicy(policy func(req *http.Request, via []*http.Request) error) Option {
	return func(client *HttpClient) {
		client.CheckRedirect = policy
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(client *HttpClient) {
		client.Timeout = timeout
	}
}

func WithCookieJar(jar http.CookieJar) Option {
	return func(client *HttpClient) {
		client.Jar = jar
	}
}

func WithoutProxy() Option {
	return func(client *HttpClient) {
		client.Transport = http.DefaultTransport
	}
}
