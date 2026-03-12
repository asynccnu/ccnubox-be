package crawler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/asynccnu/ccnubox-be/common/pkg/crypto"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/tidwall/gjson"
)

const (
	PG_URL_LOGIN_LIBRARY                        = loginCCNUPassPortURL + "?service=https%3A%2F%2Fkjyy.ccnu.edu.cn%2Frem%2Fstatic%2Fsso%2FwebOAuthRed"
	PG_URL_SEAT_LIBRARY_PREFIX                  = "https://kjyy.ccnu.edu.cn/rem/static/gotoProject/1981289670190006272"
	PG_URL_DISCUSSION_LIBRARY_PREFIX            = "https://kjyy.ccnu.edu.cn/rem/static/gotoProject/1981289795478061056"
	PG_URL_SEAT_AUTH_TOKEN_LIBRARY_PREFIX       = "https://kjyy.ccnu.edu.cn/jsq/static/public/auth/cas/"
	PG_URL_DISCUSSION_AUTH_TOKEN_LIBRARY_PREFIX = "https://kjyy.ccnu.edu.cn/spa/static/public/api/remoteCasLogin"
	PG_URL_SEST_TOKEN_CHECK                     = "https://kjyy.ccnu.edu.cn/jsq/static/frontApi/user/getUserInfo"
	PG_URL_DISCUSSION_TOKEN_CHECK               = "https://kjyy.ccnu.edu.cn/spa/static/api/book/getSchoolList"
)

type Library struct {
	Client *http.Client
	Secret string
}

type payload struct {
	LoginType string `json:"loginType"`
	Token     string `json:"token"`
}

type response struct {
	Data    interface{} `json:"data"`
	Status  bool        `json:"status"`
	Code    int         `json:"code"`
	Message string      `json:"message"`
}

func NewLibrary(client *http.Client, secret string) *Library {
	return &Library{
		Client: client,
		Secret: secret,
	}
}

// 1.LoginLibrary 使用登录通行证的client访问图书馆页面为该client设置cookie
func (c *Library) LoginLibrary(ctx context.Context) error {
	request, err := http.NewRequestWithContext(ctx, "POST", PG_URL_LOGIN_LIBRARY, nil)
	if err != nil {
		return errorx.Errorf("library: create login request failed: %w", err)
	}

	request.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36")

	resp, err := c.Client.Do(request)
	if err != nil {
		return errorx.Errorf("library: send login request failed: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

// 2.GetSeatAuthTokenFromLibrary 从图书馆系统中提取座位预约服务的 Token
func (c *Library) GetSeatAuthTokenFromLibrary(ctx context.Context) (string, error) {
	rawToken, err := c.getRawTokenFromLibrary(ctx, PG_URL_SEAT_LIBRARY_PREFIX)
	if err != nil {
		return "", errorx.Errorf("library: get raw token failed:%v", err)
	}

	rawUrl := PG_URL_SEAT_AUTH_TOKEN_LIBRARY_PREFIX + rawToken
	reqBody := payload{LoginType: "PC", Token: rawToken}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", errorx.Errorf("library: reqBody marshal failed:%v", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", rawUrl, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", errorx.Errorf("library: get auth token request failed:%v", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36")
	req.Header.Set("logintype", "PC")

	resp, err := c.Client.Do(req)
	if err != nil {
		return "", errorx.Errorf("library: send auth token request failed:%v", err)
	}
	defer resp.Body.Close()

	respBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errorx.Errorf("library: read response reqBody failed:%v", err)
	}
	var respBody response
	err = json.Unmarshal(respBodyBytes, &respBody)
	if err != nil {
		return "", errorx.Errorf("library: unmarshal resp body failed:%v", err)
	}
	if !respBody.Status || respBody.Code != 200 {
		return "", errorx.Errorf("library: interface returns error,code:%d,message:%s", respBody.Code, respBody.Message)
	}

	dataMap := respBody.Data.(map[string]interface{})
	token, ok := dataMap["token"].(string)
	if !ok || token == "" {
		return "", errorx.Errorf("library: data has format error")
	}

	return token, nil

}

// 2.CheckLibrarySeatToken 验证座位预约服务Token的有效性
func (c *Library) CheckLibrarySeatToken(ctx context.Context, token string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", PG_URL_SEST_TOKEN_CHECK, nil)
	if err != nil {
		return false, err
	}

	id, sign, ts := crypto.BuildSignWithSecret("POST", c.Secret)
	req.Header.Set("Token", token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("LoginType", "PC")
	req.Header.Set("X-Hmac-Request-Key", sign)
	req.Header.Set("X-Request-Date", fmt.Sprintf("%d", ts))
	req.Header.Set("X-Request-Id", id)

	resp, err := c.Client.Do(req)
	if err != nil {
		return false, err
	}

	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// 3.GetDiscussionAuthTokenFromLibrary 从图书馆系统中提取研讨室预约服务的 Token
func (c *Library) GetDiscussionAuthTokenFromLibrary(ctx context.Context) (string, error) {
	rawToken, err := c.getRawTokenFromLibrary(ctx, PG_URL_DISCUSSION_LIBRARY_PREFIX)
	if err != nil {
		return "", err
	}

	URL, err := url.Parse(PG_URL_DISCUSSION_AUTH_TOKEN_LIBRARY_PREFIX)
	if err != nil {
		return "", err
	}
	params := url.Values{}
	params.Set("token", rawToken)
	params.Set("noAuth", "true")
	URL.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", URL.String(), nil)
	if err != nil {
		return "", err
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	data := gjson.GetBytes(body, "data")
	if !data.Exists() {
		return "", nil
	}
	authToken := data.Get("token")
	if !authToken.Exists() {
		return "", nil
	}
	return authToken.String(), nil
}

// 4.CheckLibraryDiscussionToken 验证研讨室预约服务Token的有效性
func (c *Library) CheckLibraryDiscussionToken(ctx context.Context, token string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", PG_URL_DISCUSSION_TOKEN_CHECK, nil)
	if err != nil {
		return false, nil
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36")
	req.Header.Set("Access-Control-Allow-Origin", "*")

	resp, err := c.Client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

func (c *Library) parseRawTokenFromUrl(urlStr string) (string, error) {
	if strings.Contains(urlStr, "#") {
		parts := strings.Split(urlStr, "#")
		if len(parts) >= 2 {
			urlStr = parts[0] + parts[1]
		}
	}

	parsedUrl, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}
	queryParams := parsedUrl.Query()
	token := queryParams.Get("token")

	return token, nil
}

// 座位和研讨间通用
func (c *Library) getRawTokenFromLibrary(ctx context.Context, prefix string) (string, error) {
	rawUrl := c.buildLibraryTokenUrl(prefix)
	req, err := http.NewRequestWithContext(ctx, "GET", rawUrl, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36")
	resp, err := c.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	urlStr := resp.Request.URL.String()

	rawToken, err := c.parseRawTokenFromUrl(urlStr)
	if err != nil || rawToken == "" {
		return "", err
	}

	return rawToken, nil

}

func (c *Library) buildLibraryTokenUrl(prefix string) string {
	rand.Seed(time.Now().Unix())
	random := rand.Intn(9000) + 1000
	return fmt.Sprintf("%s?rand=%d", prefix, random)
}
