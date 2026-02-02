package crawler

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/go-kratos/kratos/v2/log"
)

var (
	INCorrectPASSWORD = errorx.New("账号密码错误")
)

func NewCrawlerClient(t time.Duration, proxyAddr string) *http.Client {
	j, _ := cookiejar.New(&cookiejar.Options{})
	netTransport := &http.Transport{
		MaxIdleConnsPerHost:   10,
		ResponseHeaderTimeout: time.Second * time.Duration(5),
	}

	client := &http.Client{
		Transport: netTransport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil
		},
	}

	client.Jar = j
	client.Timeout = t

	if proxyAddr == "" {
		return client
	}

	proxy, err := url.Parse(proxyAddr)
	if err != nil {
		log.Error("error parsing proxy addr: ", proxyAddr)
		return client
	}

	client.Transport.(*http.Transport).Proxy = http.ProxyURL(proxy)

	return client
}
