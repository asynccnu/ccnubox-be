package crawler

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/asynccnu/ccnubox-be/common/bizpkg/proxy"
)

const PG_URL = "https://bkzhjw.ccnu.edu.cn/"

func NewCrawlerClientWithCookieJar(t time.Duration, jar *cookiejar.Jar, proxyClient proxy.Client) *http.Client {
	options := []proxy.Option{
		proxy.WithRedirectPolicy(proxy.RedirectPolicyAllow),
		proxy.WithTimeout(t),
		proxy.WithProxyTransport(
			proxy.WithMaxIdleConns(10),
			proxy.WithIdleConnTimeout(90*time.Second),
			proxy.WithTLSHandshakeTimeout(10*time.Second),
		),
	}
	if jar != nil {
		options = append(options, proxy.WithCookieJar(jar))
	}
	return proxyClient.NewProxyClient(options...).Client
}

func NewJarWithCookie(targetURL, rawCookie string) *cookiejar.Jar {
	jar, _ := cookiejar.New(&cookiejar.Options{})
	// 设置目标域名
	u, err := url.Parse(targetURL)
	if err != nil {
		return nil
	}

	// 将字符串形式 Cookie 解析成 []*http.Cookie
	cookies := parseRawCookieString(rawCookie)
	jar.SetCookies(u, cookies)
	return jar
}

func parseRawCookieString(raw string) []*http.Cookie {
	parts := strings.Split(raw, ";")
	var cookies []*http.Cookie
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if kv := strings.SplitN(part, "=", 2); len(kv) == 2 {
			cookies = append(cookies, &http.Cookie{
				Name:  strings.TrimSpace(kv[0]),
				Value: strings.TrimSpace(kv[1]),
			})
		}
	}
	return cookies
}
