package proxy

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

type HttpClient struct {
	*http.Client
}

func (c *HttpClient) Use(options ...Option) {
	if len(options) == 0 {
		return
	}
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

func WithProxyTransport(options ...RoundTripperOption) Option {
	return func(client *HttpClient) {
		tr := globalProxy.NewProxyTransport()
		if globalProxy.direct {
			tr.Use(options...)
			client.Transport = tr
			return
		}

		tr.Proxy = func(req *http.Request) (*url.URL, error) {
			ctx := req.Context()
			addrs := globalProxy.GetProxyAddr(ctx, 1)
			proxyAddr := strings.TrimSpace(addrs[0])
			if proxyAddr == "" {
				return nil, nil
			}
			proxyURL, err := url.Parse(proxyAddr)
			if err != nil {
				if l := globalProxy.logger(ctx); l != nil {
					l.Warn("代理地址解析失败，fallback 到直连",
						logger.String("proxy_addr", proxyAddr),
						logger.Error(err),
					)
				}
				return nil, nil
			}
			return proxyURL, nil
		}

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
