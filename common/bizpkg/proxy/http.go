package proxy

import (
	"context"
	proxyv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/proxy/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/go-kratos/kratos/v2/log"
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

	mu sync.RWMutex
	p  proxyv1.ProxyClient
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
	addrs, err := s.getProxyAddrFromShenLong(ctx, cnt)
	if err != nil {
		log.Warn("获取神龙代理地址失败: %v", err)
		return addrs
	}
	return addrs
}

func (s *HttpProxy) update() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	res, err := s.p.GetProxyAddr(ctx, &proxyv1.GetProxyAddrRequest{})
	if err != nil {
		log.Warn("从神龙获取代理地址失败: %v", err)
		s.mu.Lock()
		s.Addr, s.AddrBackup = "", ""
		s.mu.Unlock()
		return
	}

	s.mu.Lock()
	s.Addr, s.AddrBackup = res.Addr, res.Backup
	s.mu.Unlock()
}

func NewHttpProxy(p proxyv1.ProxyClient) Client {
	globalProxy = &HttpProxy{p: p}

	globalProxy.update()
	c := cron.New()
	_, _ = c.AddFunc("@every 15s", globalProxy.update)
	c.Start()

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
