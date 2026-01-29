package crawler

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

const PG_URL = "https://bkzhjw.ccnu.edu.cn/"

func NewCrawlerClientWithCookieJar(t time.Duration, jar *cookiejar.Jar) *http.Client {
	client := &http.Client{
		Transport: nil,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil
		},
		Timeout: t,
	}
	if jar != nil {
		client.Jar = jar
	}
	return client
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
