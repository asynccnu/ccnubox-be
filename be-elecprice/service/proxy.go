package service

import (
	"context"
	proxyv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/proxy/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/robfig/cron/v3"
	"net/http"
	"net/url"
	"sync"
)

type ProxyService interface {
	StartCronTask()
}

type proxyService struct {
	ps proxyv1.ProxyClient
	l  logger.Logger

	rwMutex sync.RWMutex
	addr    string
	backup  string
}

func NewProxyService(ps proxyv1.ProxyClient, l logger.Logger) ProxyService {
	p := &proxyService{ps: ps, l: l}
	go p.fetchProxy()

	return p
}

func (p *proxyService) StartCronTask() {
	c := cron.New()
	_, _ = c.AddFunc("@every 50s", p.fetchProxy)
	c.Start()
}

func (p *proxyService) fetchProxy() {
	resp, err := p.ps.GetProxyAddr(context.Background(), &proxyv1.GetProxyAddrRequest{})
	if err != nil {
		p.l.Error("fetch proxy addr failed", logger.Error(err))
		return
	}

	p.rwMutex.Lock()
	p.addr = resp.Addr
	p.backup = resp.Backup
	p.updateClientProxy()
	p.updateBackupClientProxy()
	p.rwMutex.Unlock()
}

func (p *proxyService) updateClientProxy() {
	proxyURL, err := url.Parse(p.addr)
	if err != nil {
		p.l.Error("parse proxy url failed", logger.Error(err))
		return
	}
	clientTransport := http.DefaultTransport.(*http.Transport).Clone()
	clientTransport.Proxy = http.ProxyURL(proxyURL)
	client.Transport = clientTransport
}

func (p *proxyService) updateBackupClientProxy() {
	proxyURL, err := url.Parse(p.backup)
	if err != nil {
		p.l.Error("parse backup proxy url failed", logger.Error(err))
		return
	}
	clientTransport := http.DefaultTransport.(*http.Transport).Clone()
	clientTransport.Proxy = http.ProxyURL(proxyURL)
	clientBackup.Transport = clientTransport
}
