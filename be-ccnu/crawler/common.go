package crawler

import (
	"github.com/asynccnu/ccnubox-be/common/bizpkg/proxy"
	"net/http"
	"net/http/cookiejar"
	"time"

	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
)

var (
	INCorrectPASSWORD = errorx.New("账号密码错误")
)

func NewCrawlerClient(t time.Duration, options ...proxy.Option) *http.Client {
	j, _ := cookiejar.New(&cookiejar.Options{})
	cli := proxy.NewHttpProxyClient(
		proxy.WithProxyTransport(false),
		proxy.WithRedirectPolicy(proxy.RedirectPolicyAllow),
		proxy.WithTimeout(t),
		proxy.WithCookieJar(j),
	)
	cli.Use(options...)

	return cli.Client
}
