package crawler

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/tool"
)

const (
	LoginCCNUPassPortURL = "https://account.ccnu.edu.cn/cas/login"
)

type Passport struct {
	Client *http.Client
}

func NewPassport(client *http.Client) *Passport {
	return &Passport{
		Client: client,
	}
}

// 将放入 crawler 层，这里的组装属于行为级组装
func (c *Passport) LoginPassport(ctx context.Context, stuId string, password string) (bool, error) {
	var isInCorrectPASSWORD bool

	params, err := tool.Retry(func() (*accountRequestParams, error) {
		return c.getParamsFromHtml(ctx)
	})
	if err != nil {
		return false, errorx.Errorf("passport: get login params failed: %w", err)
	}

	_, err = tool.Retry(func() (string, error) {
		err := c.loginCCNUPassport(ctx, stuId, password, params)
		if errorx.Is(err, INCorrectPASSWORD) {
			isInCorrectPASSWORD = true
			return "", nil
		}
		return "", err
	})

	if isInCorrectPASSWORD {
		return false, INCorrectPASSWORD
	}

	if err != nil {
		return false, errorx.Errorf("passport: login failed: %w", err)
	}

	return true, nil
}

// 1. 前置请求：从 HTML 中提取参数
func (c *Passport) getParamsFromHtml(ctx context.Context) (*accountRequestParams, error) {
	params := &accountRequestParams{}

	request, err := http.NewRequestWithContext(ctx, "GET", LoginCCNUPassPortURL, nil)
	if err != nil {
		return params, errorx.Errorf("create login request failed: %w", err)
	}

	resp, err := c.Client.Do(request)
	if err != nil {
		return params, errorx.Errorf("send login request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return params, errorx.Errorf("read login html failed: %w", err)
	}

	var JSESSIONID string
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "JSESSIONID" {
			JSESSIONID = cookie.Value
			break
		}
	}
	if JSESSIONID == "" {
		return params, errorx.Errorf("parse cookie failed: missing JSESSIONID")
	}

	bodyStr := string(body)

	lt, err := extractField(bodyStr, `name="lt".+value="(.+)"`, "lt")
	if err != nil {
		return params, err
	}

	execution, err := extractField(bodyStr, `name="execution".+value="(.+)"`, "execution")
	if err != nil {
		return params, err
	}

	_eventId, err := extractField(bodyStr, `name="_eventId".+value="(.+)"`, "_eventId")
	if err != nil {
		return params, err
	}

	params.lt = lt
	params.execution = execution
	params._eventId = _eventId
	params.submit = "LOGIN"
	params.JSESSIONID = JSESSIONID

	return params, nil
}

// 2. 登录 CCNU 通行证
func (c *Passport) loginCCNUPassport(
	ctx context.Context,
	studentId string,
	password string,
	params *accountRequestParams,
) error {

	v := url.Values{}
	v.Set("username", studentId)
	v.Set("password", password)
	v.Set("lt", params.lt)
	v.Set("execution", params.execution)
	v.Set("_eventId", params._eventId)
	v.Set("submit", params.submit)

	urlstr := LoginCCNUPassPortURL + ";jsessionid=" + params.JSESSIONID
	request, err := http.NewRequestWithContext(ctx, "POST", urlstr, strings.NewReader(v.Encode()))
	if err != nil {
		return errorx.Errorf("create login post request failed: %w", err)
	}

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := c.Client.Do(request)
	if err != nil {
		return errorx.Errorf("send login post request failed: %w", err)
	}
	defer resp.Body.Close()

	res, err := io.ReadAll(resp.Body)
	if err != nil {
		return errorx.Errorf("read login response failed: %w", err)
	}

	if strings.Contains(string(res), "您输入的用户名或密码有误") {
		return INCorrectPASSWORD
	}

	if resp.Header.Get("Set-Cookie") == "" {
		return errorx.Errorf("login failed: missing Set-Cookie")
	}

	return nil
}

// HTML 字段提取工具
func extractField(body, pattern, name string) (string, error) {
	reg := regexp.MustCompile(pattern)
	arr := reg.FindStringSubmatch(body)
	if len(arr) != 2 {
		return "", errorx.Errorf("parse html failed: missing %s", name)
	}
	return arr[1], nil
}

type accountRequestParams struct {
	lt         string
	execution  string
	_eventId   string
	submit     string
	JSESSIONID string
}
