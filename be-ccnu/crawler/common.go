package crawler

import (
	"github.com/asynccnu/ccnubox-be/common/bizpkg/proxy"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"net/http"
	"net/http/cookiejar"
	"time"
)

var (
	INCorrectPASSWORD = errorx.New("账号密码错误")
)

func NewCrawlerClient(pc proxy.Client, t time.Duration, options ...proxy.Option) *http.Client {
	j, _ := cookiejar.New(&cookiejar.Options{})
	opts := []proxy.Option{
		proxy.WithProxyTransport(),
		proxy.WithRedirectPolicy(proxy.RedirectPolicyAllow),
		proxy.WithTimeout(t),
		proxy.WithCookieJar(j),
	}
	opts = append(opts, options...)
	return pc.NewProxyClient(opts...).Client
}
