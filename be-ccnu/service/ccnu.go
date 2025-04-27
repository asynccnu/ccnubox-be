package service

import (
	"context"
	"errors"
	"fmt"
	ccnuv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/ccnu/v1"
	"github.com/asynccnu/ccnubox-be/be-ccnu/pkg/errorx"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
)

// 定义错误,这里将kratos的error作为一个重要部分传入,此处的错误并不直接在service中去捕获,而是选择在更底层的爬虫去捕获,因为爬虫的错误处理非常复杂
var (
	CCNUSERVER_ERROR = func(err error) error {
		return errorx.New(ccnuv1.ErrorCcnuserverError("ccnu服务器错误"), "ccnuServer", err)
	}

	Invalid_SidOrPwd_ERROR = func(err error) error {
		return errorx.New(ccnuv1.ErrorInvalidSidOrPwd("账号密码错误"), "user", err)
	}

	SYSTEM_ERROR = func(err error) error {
		return errorx.New(ccnuv1.ErrorSystemError("系统内部错误"), "system", err)
	}
)

func (c *ccnuService) GetCCNUCookie(ctx context.Context, studentId string, password string) (string, error) {

	//初始化client
	client := c.client()

	//从ccnu主页获取相关参数
	params, err := c.makeAccountPreflightRequest(client)
	if err != nil {
		return "", err
	}

	//登陆ccnu通行证
	client, err = c.loginClient(ctx, client, studentId, password, params)
	if err != nil {
		return "", err
	}

	//登陆本科生院
	client, err = c.xkLoginClient(client)
	if err != nil {
		return "", err
	}

	//解析获取cookie
	cookie, err := c.getBKSCookie(client)
	if err != nil {
		return "", err
	}

	return cookie, nil
}

func (c *ccnuService) Login(ctx context.Context, studentId string, password string) (bool, error) {
	client := c.client()
	//从ccnu主页获取相关参数
	params, err := c.makeAccountPreflightRequest(client)
	if err != nil {
		return false, err
	}

	client, err = c.loginClient(ctx, client, studentId, password, params)
	return client != nil, err
}

// getBKSCookie 返回本科生院的Cookie
func (c *ccnuService) getBKSCookie(client *http.Client) (string, error) {

	// 构造用于获取 Cookie 的 URL（即请求时的登录 URL）
	loginURL := "http://xk.ccnu.edu.cn/jwglxt"
	u, err := url.Parse(loginURL)
	if err != nil {
		return "", SYSTEM_ERROR(err)
	}

	// 从 CookieJar 中获取这个域名的 Cookie
	cookies := client.Jar.Cookies(u)
	if len(cookies) == 0 {
		return "", SYSTEM_ERROR(errors.New("no cookies found after login"))
	}

	// 拼接所有 Cookie 为字符串格式
	var cookieStr string
	for _, cookie := range cookies {
		if cookie.Name == "JSESSIONID" {
			cookieStr = cookie.Value
		}
	}

	// 返回拼接好的 Cookie 字符串
	return fmt.Sprintf("JSESSIONID=%s", cookieStr), nil

}

// 2.xkLoginClient 教务系统模拟登录
func (c *ccnuService) xkLoginClient(client *http.Client) (*http.Client, error) {

	//华师本科生院登陆
	request, err := http.NewRequest("GET", "https://account.ccnu.edu.cn/cas/login?service=http%3A%2F%2Fxk.ccnu.edu.cn%2Fsso%2Fpziotlogin", nil)
	if err != nil {
		return nil, CCNUSERVER_ERROR(err)
	}

	resp, err := client.Do(request)
	if err != nil {
		return nil, CCNUSERVER_ERROR(err)
	}
	defer resp.Body.Close()

	return client, nil
}

// 1.登陆ccnu通行证
func (c *ccnuService) loginClient(ctx context.Context, client *http.Client, studentId string, password string, params *accountRequestParams) (*http.Client, error) {

	v := url.Values{}
	v.Set("username", studentId)
	v.Set("password", password)
	v.Set("lt", params.lt)
	v.Set("execution", params.execution)
	v.Set("_eventId", params._eventId)
	v.Set("submit", params.submit)

	request, err := http.NewRequest("POST", "https://account.ccnu.edu.cn/cas/login;jsessionid="+params.JSESSIONID, strings.NewReader(v.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/72.0.3626.109 Safari/537.36")
	request.WithContext(ctx)
	//创建一个带jar的客户端
	j, _ := cookiejar.New(&cookiejar.Options{})
	client.Jar = j
	//发送请求
	resp, err := client.Do(request)
	if err != nil {
		var opErr *net.OpError
		if errors.As(err, &opErr) {
			return nil, CCNUSERVER_ERROR(err)
		}
		return nil, err
	}
	defer resp.Body.Close()

	if len(resp.Header.Get("Set-Cookie")) == 0 {
		return nil, Invalid_SidOrPwd_ERROR(errors.New("学号或密码错误"))
	}
	return client, nil
}

type accountRequestParams struct {
	lt         string
	execution  string
	_eventId   string
	submit     string
	JSESSIONID string
}

// 0.前置请求,从html中提取相关参数
func (c *ccnuService) makeAccountPreflightRequest(client *http.Client) (*accountRequestParams, error) {
	var JSESSIONID string
	var lt string
	var execution string
	var _eventId string

	params := &accountRequestParams{}

	// 初始化 http request
	request, err := http.NewRequest("GET", "https://account.ccnu.edu.cn/cas/login", nil)
	if err != nil {
		return params, SYSTEM_ERROR(err)
	}

	// 发起请求
	resp, err := client.Do(request)
	if err != nil {
		return params, SYSTEM_ERROR(err)
	}
	defer resp.Body.Close()

	// 读取 MsgContent
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return params, SYSTEM_ERROR(err)
	}

	// 获取 Cookie 中的 JSESSIONID
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "JSESSIONID" {
			JSESSIONID = cookie.Value
		}
	}

	if JSESSIONID == "" {
		return params, SYSTEM_ERROR(errors.New("Can not get JSESSIONID"))
	}

	// 正则匹配 HTML 返回的表单字段
	ltReg := regexp.MustCompile("name=\"lt\".+value=\"(.+)\"")
	executionReg := regexp.MustCompile("name=\"execution\".+value=\"(.+)\"")
	_eventIdReg := regexp.MustCompile("name=\"_eventId\".+value=\"(.+)\"")

	bodyStr := string(body)

	ltArr := ltReg.FindStringSubmatch(bodyStr)
	if len(ltArr) != 2 {
		return params, CCNUSERVER_ERROR(errors.New("Can not get lt"))
	}
	lt = ltArr[1]

	execArr := executionReg.FindStringSubmatch(bodyStr)
	if len(execArr) != 2 {
		return params, CCNUSERVER_ERROR(errors.New("Can not get execution"))
	}

	execution = execArr[1]

	_eventIdArr := _eventIdReg.FindStringSubmatch(bodyStr)
	if len(_eventIdArr) != 2 {
		return params, CCNUSERVER_ERROR(errors.New("Can not get _eventId"))
	}
	_eventId = _eventIdArr[1]

	params.lt = lt
	params.execution = execution
	params._eventId = _eventId
	params.submit = "LOGIN"
	params.JSESSIONID = JSESSIONID

	return params, nil
}

// -1.前置工作,用于初始化client
func (c *ccnuService) client() *http.Client {
	j, _ := cookiejar.New(&cookiejar.Options{})
	return &http.Client{
		Transport: nil,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil
		},
		Jar:     j,
		Timeout: c.timeout,
	}
}
