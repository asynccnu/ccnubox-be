package crawler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

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

// 将放入crawler层，这里的组装属于行为级组装，不用移动至服务级
func (c *Passport) LoginPassport(ctx context.Context, stuId string, password string) (bool, error) {
	var (
		isInCorrectPASSWORD = false //用于判断是否是账号密码错误
	)

	params, err := tool.Retry(func() (*accountRequestParams, error) {
		return c.getParamsFromHtml(ctx)
	})
	if err != nil {
		return false, err
	}

	//此处比较特殊由于账号密码错误是必然无效的请求,应当直接返回
	_, err = tool.Retry(func() (string, error) {
		err := c.loginCCNUPassport(ctx, stuId, password, params)
		if errors.Is(err, INCorrectPASSWORD) {
			// 标识账号密码错误,强制结束
			isInCorrectPASSWORD = true
			return "", nil
		}
		return "", err
	})
	//如果密码有误
	if isInCorrectPASSWORD {
		return false, INCorrectPASSWORD
	}
	//如果存在错误
	if err != nil {
		return false, err
	}
	return true, nil
}

// 1.前置请求，从html中提取相关参数
func (c *Passport) getParamsFromHtml(ctx context.Context) (*accountRequestParams, error) {
	var JSESSIONID string
	var lt string
	var execution string
	var _eventId string

	params := &accountRequestParams{}

	// 初始化 http request
	request, err := http.NewRequestWithContext(ctx, "GET", loginCCNUPassPortURL, nil)
	if err != nil {
		return params, err
	}

	// 发起请求
	resp, err := c.Client.Do(request)
	if err != nil {
		return params, err
	}
	defer resp.Body.Close()

	// 读取 MsgContent
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return params, err
	}

	// 获取 Cookie 中的 JSESSIONID
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "JSESSIONID" {
			JSESSIONID = cookie.Value
		}
	}

	if JSESSIONID == "" {
		return params, errors.New("Can not get JSESSIONID")
	}

	// 正则匹配 HTML 返回的表单字段
	ltReg := regexp.MustCompile("name=\"lt\".+value=\"(.+)\"")
	executionReg := regexp.MustCompile("name=\"execution\".+value=\"(.+)\"")
	_eventIdReg := regexp.MustCompile("name=\"_eventId\".+value=\"(.+)\"")

	bodyStr := string(body)

	ltArr := ltReg.FindStringSubmatch(bodyStr)
	if len(ltArr) != 2 {
		return params, errors.New("Can not get lt")
	}
	lt = ltArr[1]

	execArr := executionReg.FindStringSubmatch(bodyStr)
	if len(execArr) != 2 {
		return params, errors.New("Can not get execution")
	}
	execution = execArr[1]

	_eventIdArr := _eventIdReg.FindStringSubmatch(bodyStr)
	if len(_eventIdArr) != 2 {
		return params, errors.New("Can not get _eventId")
	}
	_eventId = _eventIdArr[1]

	params.lt = lt
	params.execution = execution
	params._eventId = _eventId
	params.submit = "LOGIN"
	params.JSESSIONID = JSESSIONID

	return params, nil
}

// 2.登陆ccnu通行证
func (c *Passport) loginCCNUPassport(ctx context.Context, studentId string, password string, params *accountRequestParams) error {
	v := url.Values{}
	v.Set("username", studentId)
	v.Set("password", password)
	v.Set("lt", params.lt)
	v.Set("execution", params.execution)
	v.Set("_eventId", params._eventId)
	v.Set("submit", params.submit)

	urlstr := loginCCNUPassPortURL + ";jsessionid=" + params.JSESSIONID
	request, err := http.NewRequestWithContext(ctx, "POST", urlstr, strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/72.0.3626.109 Safari/537.36")

	resp, err := c.Client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	res, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if strings.Contains(string(res), "您输入的用户名或密码有误") {
		return INCorrectPASSWORD
	}

	if len(resp.Header.Get("Set-Cookie")) == 0 {
		return errors.New("登录失败，未返回 Cookie")
	}

	return nil
}

type accountRequestParams struct {
	lt         string
	execution  string
	_eventId   string
	submit     string
	JSESSIONID string
}
