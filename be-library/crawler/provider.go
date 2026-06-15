package crawler

import (
	"net/http"
	"time"

	"github.com/asynccnu/ccnubox-be/common/bizpkg/proxy"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

func InitCrawlerHttpClient(p proxy.Client) *http.Client {
	cli := p.NewProxyClient(
		proxy.WithProxyTransport(),
		proxy.WithTimeout(10*time.Second),
	)
	return cli.Client
}

func NewLibraryCrawlerMust(client *http.Client, l logger.Logger, secret string) *Crawler {
	c, err := NewLibraryCrawler(client, l, secret)
	if err != nil {
		panic(err)
	}
	return c
}
