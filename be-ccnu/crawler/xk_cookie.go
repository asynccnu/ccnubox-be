package crawler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/pkg/httpx"
	"golang.org/x/net/html"
)

const maxXKProbeBody = 1 << 20

// CheckCookie 复用 be-grade/crawler/undergrad.go 的成绩响应契约，只取一条记录。
func (c *UnderGrad) CheckCookie(ctx context.Context, cookie string) (bool, error) {
	const endpoint = "https://bkzhjw.ccnu.edu.cn/jsxsd/kscj/cjcx_list?pageNum=1&pageSize=1&kksj=&kcxz=&kcsx=&kcmc=&xsfs=all&sfxsbcxq=1"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false, err
	}
	return checkXKCookie(c.Client, req, cookie, false)
}

// CheckCookie 复用 be-grade/crawler/postgraduate.go 的 items 数组契约，不猜测研究生首页地址。
func (c *PostGraduate) CheckCookie(ctx context.Context, cookie string) (bool, error) {
	const endpoint = postgraduateURL + "/yjsxt/cjcx/cjcx_cxDgXscj.html?doType=query&gnmkdm=N305005"
	form := url.Values{
		"xnm": {""}, "xqm": {""}, "cjzt": {"3"}, "_search": {"false"},
		"queryModel.showCount": {"1"}, "queryModel.currentPage": {"1"},
		"queryModel.sortName": {""}, "queryModel.sortOrder": {"asc"}, "time": {"1"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return checkXKCookie(c.client, req, cookie, true)
}

func checkXKCookie(client *http.Client, req *http.Request, cookie string, postgraduate bool) (bool, error) {
	if cookie == "" {
		return false, nil
	}
	// 保留已有 Transport、代理与超时，禁止探测请求自动跳转或复用登录 CookieJar。
	probe := *client
	probe.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	probe.Jar = nil
	req.Header.Set("Cookie", cookie)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	resp, err := probe.Do(req)
	if err != nil {
		return false, errorx.Errorf("xk cookie probe request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		location, err := resp.Location()
		if err == nil && isXKLoginURL(location) {
			return false, nil
		}
		return false, errorx.Errorf("xk cookie probe unexpected redirect, status: %d", resp.StatusCode)
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return false, nil
	}
	// 403 可能来自代理、WAF 或权限拦截，不能当作认证过期触发重新登录。
	if resp.StatusCode != http.StatusOK {
		return false, errorx.Errorf("xk cookie probe unexpected status: %d", resp.StatusCode)
	}
	body, err := httpx.ReadResponse(resp, httpx.WithMaxBodyBytes(maxXKProbeBody))
	if err != nil {
		return false, errorx.Errorf("xk cookie probe read response failed: %w", err)
	}
	if isXKLoginPage(body) {
		return false, nil
	}

	// 必须存在已知成功结构，空 JSON、错误页和未知业务码均视为不可确定。
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		return false, errorx.New("xk cookie probe unrecognized response")
	}
	field := "data"
	if postgraduate {
		field = "items"
	} else {
		var code *int
		if err := json.Unmarshal(payload["code"], &code); err != nil || code == nil || *code != 0 {
			return false, errorx.New("xk cookie probe unrecognized business status")
		}
	}
	data := bytes.TrimSpace(payload[field])
	if len(data) == 0 || data[0] != '[' {
		return false, errorx.New("xk cookie probe missing result array")
	}
	return true, nil
}

func isXKLoginURL(u *url.URL) bool {
	path := strings.ToLower(strings.TrimRight(u.Path, "/"))
	return (strings.EqualFold(u.Hostname(), "account.ccnu.edu.cn") && path == "/cas/login") ||
		(strings.EqualFold(u.Hostname(), "bkzhjw.ccnu.edu.cn") && path == "/jsxsd/xk/login") ||
		(strings.EqualFold(u.Hostname(), "grd.ccnu.edu.cn") &&
			(path == "/yjsxt/xtgl/login_slogin.html" || path == "/yjsxt/xtgl/login_login.html"))
}

func isXKLoginPage(body []byte) bool {
	text := strings.ToLower(strings.TrimSpace(string(body)))
	if !strings.HasPrefix(text, "<") {
		return false
	}
	// 本科系统未登录时会返回 200 + JS 跳转；研究生登录表单使用 yhm/mm。
	if strings.Contains(text, "https://account.ccnu.edu.cn/cas/login") ||
		strings.Contains(text, "/yjsxt/xtgl/login_slogin.html") {
		return true
	}
	tokens := html.NewTokenizer(strings.NewReader(text))
	var form, username, password bool
	for {
		switch tokens.Next() {
		case html.ErrorToken:
			return form && username && password
		case html.StartTagToken, html.SelfClosingTagToken:
			token := tokens.Token()
			if token.Data == "form" {
				form = true
			}
			if token.Data != "input" {
				continue
			}
			for _, attr := range token.Attr {
				if attr.Key != "name" && attr.Key != "id" {
					continue
				}
				switch attr.Val {
				case "yhm", "username":
					username = true
				case "mm", "password":
					password = true
				}
			}
		}
	}
}
