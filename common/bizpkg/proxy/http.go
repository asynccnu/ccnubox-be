package proxy

import (
	"context"
	proxyv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/proxy/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/robfig/cron/v3"
	"net/http"
	"net/http/cookiejar"
	"sync"
	"time"
)

var (
	globalProxy *HttpProxy
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

func (s *HttpProxy) getProxyAddrFromShenLong(_ context.Context, cnt int) ([]string, error) {
	addrs := make([]string, cnt)
	switch cnt {
	case 1:
		s.mu.RLock()
		addrs[0] = s.Addr
		s.mu.RUnlock()
	case 2:
		s.mu.RLock()
		addrs[0], addrs[1] = s.Addr, s.AddrBackup
		s.mu.RUnlock()
	default:
		return addrs, errorx.New("不支持的代理地址数量")
	}
	return addrs, nil
}

func (s *HttpProxy) GetProxyAddr(ctx context.Context, cnt int) []string {
	if s.direct {
		return make([]string, cnt)
	}

	addrs, err := s.getProxyAddrFromShenLong(ctx, cnt)
	if err != nil {
		if l := s.logger(ctx); l != nil {
			l.Warn("获取缓存代理地址失败", logger.Error(err))
		}
		return addrs
	}
	return addrs
}

func (s *HttpProxy) update() {
	if s.direct {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	res, err := s.p.GetProxyAddr(ctx, &proxyv1.GetProxyAddrRequest{})
	if err != nil {
		if l := s.logger(ctx); l != nil {
			l.Warn("从 be-proxy 获取代理地址失败", logger.Error(err))
		}
		s.mu.Lock()
		s.Addr, s.AddrBackup = "", ""
		s.mu.Unlock()
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
