package crawler

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const (
	loginCCNUPassPortURL = "https://account.ccnu.edu.cn/cas/login"

	NewCASURL = loginCCNUPassPortURL + "?service=https://bkzhjw.ccnu.edu.cn/jsxsd/framework/xsMainV.htmlx"
	NewpgUrl  = "https://bkzhjw.ccnu.edu.cn/jsxsd/"
	CASURL    = loginCCNUPassPortURL + "?service=http%3A%2F%2Fxk.ccnu.edu.cn%2Fsso%2Fpziotlogin"
	pgUrl     = "http://xk.ccnu.edu.cn/jwglxt"

	old  = "old"
	new_ = "new"
)

// 存放本科生院相关的爬虫
type UnderGrad struct {
	Client *http.Client
}

func NewUnderGrad(client *http.Client) *UnderGrad {
	return &UnderGrad{
		Client: client,
	}
}

// 1.LoginPUnderGradSystem 教务系统模拟登录
func (c *UnderGrad) LoginUnderGradSystem(ctx context.Context, tpe ...string) error {
	finalUrl := CASURL //默认用老的
	if len(tpe) != 0 && tpe[0] == new_ {
		finalUrl = NewCASURL
	}

	request, err := http.NewRequestWithContext(ctx, "POST", finalUrl, nil)
	if err != nil {
		return err
	}
	request.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/72.0.3626.109 Safari/537.36")

	resp, err := c.Client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// 2.GetCookieFromUnderGradSystem 从教务系统中提取Cookie
func (c *UnderGrad) GetCookieFromUnderGradSystem(tpe ...string) (string, error) {
	finalUrl := pgUrl //默认用老的
	if len(tpe) != 0 && tpe[0] == new_ {
		finalUrl = NewpgUrl
	}

	parsedURL, err := url.Parse(finalUrl)
	if err != nil {
		return "", fmt.Errorf("解析 URL 出错: %v", err)
	}

	cookies := c.Client.Jar.Cookies(parsedURL)

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
