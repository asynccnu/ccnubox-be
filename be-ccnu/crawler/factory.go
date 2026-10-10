package crawler

import (
	"time"

	"github.com/asynccnu/ccnubox-be/be-ccnu/service"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/proxy"
)

// Factory 为每次 service 操作创建新的 CCNUCrawler。
// Crawler 在每条需要认证的流程中再创建独立的 HTTP client 和 CookieJar。
type Factory struct {
	proxyClient proxy.Client
	timeout     time.Duration
	secret      string
}

func NewFactory(proxyClient proxy.Client, secret string) *Factory {
	return &Factory{
		proxyClient: proxyClient,
		timeout:     2 * time.Minute,
		secret:      secret,
	}
}

func (f *Factory) New() service.Crawler {
	return NewCCNUCrawler(f.proxyClient, f.timeout, f.secret)
}

var _ service.CrawlerFactory = (*Factory)(nil)
