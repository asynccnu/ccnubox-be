package crawler

import (
	"net/http"
	"net/http/cookiejar"
	"time"

	"github.com/asynccnu/ccnubox-be/common/bizpkg/proxy"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
)

var INCorrectPASSWORD = errorx.New("账号密码错误")

func newCrawlerClient(pc proxy.Client, t time.Duration, options ...proxy.Option) *http.Client {
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
