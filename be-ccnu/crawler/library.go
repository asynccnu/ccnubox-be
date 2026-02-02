package crawler

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
)

const (
	PG_URL_LIBRARY = "http://kjyy.ccnu.edu.cn/ClientWeb/default.aspx"
)

type Library struct {
	Client *http.Client
}

func NewLibrary(client *http.Client) *Library {
	return &Library{
		Client: client,
	}
}

// 1.LoginLibrary 使用登录通行证的client访问图书馆页面为该client设置cookie
func (c *Library) LoginLibrary(ctx context.Context) error {
	request, err := http.NewRequestWithContext(ctx, "GET", PG_URL_LIBRARY, nil)
	if err != nil {
		return errorx.Errorf("library: create login request failed: %w", err)
	}

	request.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/72.0.3626.109 Safari/537.36")

	resp, err := c.Client.Do(request)
	if err != nil {
		return errorx.Errorf("library: send login request failed: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

// 2.GetCookieFromLibrarySystem 从图书馆系统中提取 Cookie
func (c *Library) GetCookieFromLibrarySystem() (string, error) {
	parsedURL, err := url.Parse(PG_URL_LIBRARY)
	if err != nil {
		return "", errorx.Errorf("library: parse url failed: %w", err)
	}

	// 检查 Jar 是否为空，防止空指针异常
	if c.Client.Jar == nil {
		return "", errorx.Errorf("library: cookie jar is nil")
	}

	cookies := c.Client.Jar.Cookies(parsedURL)
	if len(cookies) == 0 {
		return "", errorx.Errorf("library: no cookies found for %s", PG_URL_LIBRARY)
	}

	var cookieStr strings.Builder
	for i, cookie := range cookies {
		cookieStr.WriteString(cookie.Name)
		cookieStr.WriteString("=")
		cookieStr.WriteString(cookie.Value)
		if i != len(cookies)-1 {
			cookieStr.WriteString("; ")
		}
	}

	return cookieStr.String(), nil
}
