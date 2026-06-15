package proxy

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"sync"
	"time"

	proxyv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/proxy/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/robfig/cron/v3"
)

var (
	globalProxy = new(HttpProxy)
)

func GlobalHttpProxyClient() Client {
	return globalProxy
}

func GlobalHttpProxyGetter() Getter {
	return globalProxy
}

func GlobalHttpProxyTransporter() Transporter {
	return globalProxy
}

type HttpProxy struct {
	Addr       string
	AddrBackup string

	direct bool
	mu     sync.RWMutex
	p      proxyv1.ProxyClient
	l      logger.Logger
}

func (s *HttpProxy) logger(ctx context.Context) logger.Logger {
	if s.l != nil {
		return s.l.WithContext(ctx)
	}
	return nil
}

func (s *HttpProxy) GetProxyAddr(_ context.Context, cnt int) []string {
	if s.direct {
		return make([]string, cnt)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	addrs := make([]string, cnt)
	for i := 0; i < cnt; i++ {
		if i == 0 {
			addrs[i] = s.Addr
		} else {
			addrs[i] = s.AddrBackup
		}
		if addrs[i] == "" && i > 0 {
			addrs[i] = addrs[0]
		}
	}
	return addrs
}

func (s *HttpProxy) update() {
	if s.direct {
		return
	}
	if s.p == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	res, err := s.p.GetProxyAddr(ctx, &proxyv1.GetProxyAddrRequest{})
	if err != nil {
		if l := s.logger(ctx); l != nil {
			l.Warn("从 be-proxy 获取代理地址失败", logger.Error(err))
		}
		return
	}

	s.mu.Lock()
	s.Addr, s.AddrBackup = res.Addr, res.Backup
	s.mu.Unlock()
}

func NewHttpProxy(p proxyv1.ProxyClient, l logger.Logger) Client {
	globalProxy = &HttpProxy{p: p, l: l}

	globalProxy.update()
	c := cron.New()
	_, _ = c.AddFunc("@every 15s", globalProxy.update)
	c.Start()

	return globalProxy
}

func NewDirectHttpProxy(l logger.Logger) Client {
	globalProxy = &HttpProxy{direct: true, l: l}
	if l != nil {
		l.Warn("proxy direct 模式已启用，请求将绕过 be-proxy")
	}
	return globalProxy
}

func (s *HttpProxy) NewProxyClient(ops ...Option) *HttpClient {
	jar, _ := cookiejar.New(&cookiejar.Options{})
	c := &HttpClient{&http.Client{
		Timeout:       time.Second * 10,
		CheckRedirect: RedirectPolicyDeny,
		Jar:           jar,
	},
	}

	for _, op := range ops {
		op(c)
	}

	return c
}

func (s *HttpProxy) NewProxyTransport(ops ...RoundTripperOption) *HttpTransport {
	tr := &HttpTransport{&http.Transport{
		IdleConnTimeout: 30 * time.Second,
		MaxConnsPerHost: 70,
		MaxIdleConns:    70,
	}}

	for _, op := range ops {
		op(tr)
	}

	return tr
}

func NewHttpProxyClient(ops ...Option) *HttpClient {
	return globalProxy.NewProxyClient(ops...)
}

func NewHttpProxyTransport(ops ...RoundTripperOption) *HttpTransport {
	return globalProxy.NewProxyTransport(ops...)
}
