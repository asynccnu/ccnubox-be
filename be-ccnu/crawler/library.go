package crawler

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/asynccnu/ccnubox-be/common/pkg/crypto"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/pkg/httpx"
	"golang.org/x/sync/singleflight"
)

const (
	PG_URL_LOGIN_LIBRARY                        = loginCCNUPassPortURL + "?service=https%3A%2F%2Fkjyy.ccnu.edu.cn%2Frem%2Fstatic%2Fsso%2FwebOAuthRed"
	PG_URL_SEAT_LIBRARY_PREFIX                  = "https://kjyy.ccnu.edu.cn/rem/static/gotoProject/1981289670190006272"
	PG_URL_DISCUSSION_LIBRARY_PREFIX            = "https://kjyy.ccnu.edu.cn/rem/static/gotoProject/1981289795478061056"
	PG_URL_SEAT_AUTH_TOKEN_LIBRARY_PREFIX       = "https://kjyy.ccnu.edu.cn/jsq/static/public/auth/cas/"
	PG_URL_DISCUSSION_AUTH_TOKEN_LIBRARY_PREFIX = "https://kjyy.ccnu.edu.cn/spa/static/public/api/remoteCasLogin"
	PG_URL_SEST_TOKEN_CHECK                     = "https://kjyy.ccnu.edu.cn/jsq/static/frontApi/user/getUserInfo"
	PG_URL_SEAT_SYSTEM_CONFIG                   = "https://kjyy.ccnu.edu.cn/jsq/static/public/cg/getSysSet/PC"
	PG_URL_DISCUSSION_TOKEN_CHECK               = "https://kjyy.ccnu.edu.cn/spa/static/api/book/getSchoolList"
)

type Library struct {
	Client *http.Client
	Secret string
}

const (
	seatHMACRequestTimeout = 5 * time.Second
	seatDynamicKeyTTL      = 10 * time.Minute
	seatFallbackKeyTTL     = 30 * time.Second
)

var seatHMACCache struct {
	sync.RWMutex
	key   string
	until time.Time
}

var seatHMACFetchGroup singleflight.Group

type payload struct {
	LoginType string `json:"loginType"`
	Token     string `json:"token"`
}

type response struct {
	Data    json.RawMessage `json:"data"`
	Status  bool            `json:"status"`
	Code    int             `json:"code"`
	Message string          `json:"message"`
}

func NewLibrary(client *http.Client, secret string) *Library {
	return &Library{
		Client: client,
		Secret: secret,
	}
}

// LoginLibrary 使用登录通行证的client访问图书馆页面为该client设置cookie
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
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return errorx.Errorf("library: login returned HTTP %d", resp.StatusCode)
	}

	return nil
}

// GetSeatAuthTokenFromLibrary 从图书馆系统中提取座位预约服务的 Token
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
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("LoginType", "PC")

	resp, err := c.Client.Do(req)
	if err != nil {
		return "", errorx.Errorf("library: send auth token request failed:%v", err)
	}
	defer resp.Body.Close()

	respBodyBytes, err := httpx.ReadResponse(resp)
	if err != nil {
		return "", errorx.Errorf("library: read response reqBody failed:%v", err)
	}
	respBody, err := decodeLibraryResponse(resp, respBodyBytes)
	if err != nil {
		return "", err
	}
	token, err := decodeSeatAuthToken(respBody.Data)
	if err != nil {
		return "", errorx.Errorf("library: data has format error")
	}

	return token, nil

}

// decodeSeatAuthToken 兼容新旧座位服务的响应格式。新版直接返回 token 字符串，
// 旧版返回 {"token":"..."}，同时兼容两者可保证学校侧迁移期间 be-user 的
// token 缓存流程正常工作。
func decodeSeatAuthToken(raw json.RawMessage) (string, error) {
	var direct string
	if err := json.Unmarshal(raw, &direct); err == nil && strings.TrimSpace(direct) != "" {
		return direct, nil
	}
	var legacy struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(raw, &legacy); err != nil || strings.TrimSpace(legacy.Token) == "" {
		return "", errors.New("seat auth response did not contain a token")
	}
	return legacy.Token, nil
}

// CheckLibrarySeatToken 验证座位预约服务Token的有效性
func (c *Library) CheckLibrarySeatToken(ctx context.Context, token string) (bool, error) {
	secret, dynamicErr := c.getSeatHMACKey(ctx, token, false)
	if dynamicErr == nil {
		valid, err := c.checkLibrarySeatTokenWithSecret(ctx, token, secret)
		if err != nil || valid {
			return valid, err
		}
		// 系统级密钥可能在缓存过期前轮换，因此在判定用户 token 无效前刷新一次。
		refreshed, refreshErr := c.getSeatHMACKey(ctx, token, true)
		if refreshErr == nil && refreshed != secret {
			return c.checkLibrarySeatTokenWithSecret(ctx, token, refreshed)
		}
		return false, nil
	}
	// 兼容尚未提供 getSysSet/PC 的旧版学校服务，动态配置仍为权威数据源。
	if strings.TrimSpace(c.Secret) == "" {
		return false, dynamicErr
	}
	return c.checkLibrarySeatTokenWithSecret(ctx, token, c.Secret)
}

func (c *Library) checkLibrarySeatTokenWithSecret(ctx context.Context, token, secret string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", PG_URL_SEST_TOKEN_CHECK, nil)
	if err != nil {
		return false, err
	}

	id, sign, ts := crypto.BuildSignWithSecret("POST", secret)
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
	// HTTP 鉴权拒绝也可能由系统签名密钥轮换引起，交由上层刷新一次后再判定 token。
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return false, nil
	}
	body, err := httpx.ReadResponse(resp)
	if err != nil {
		return false, err
	}
	_, err = decodeLibraryResponse(resp, body)
	return err == nil, nil
}

func (c *Library) getSeatHMACKey(ctx context.Context, token string, force bool) (string, error) {
	if !force {
		if key, ok := cachedSeatHMACKey(); ok {
			return key, nil
		}
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	resultCh := seatHMACFetchGroup.DoChan("seat_hmac_key", func() (interface{}, error) {
		// 等待合并期间缓存可能已由上一轮请求填充，非强制刷新时再次检查。
		if !force {
			if key, ok := cachedSeatHMACKey(); ok {
				return key, nil
			}
		}

		// 共享请求不继承任一调用者的取消信号，避免首个调用者取消连累其他等待者。
		requestCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), seatHMACRequestTimeout)
		defer cancel()
		key, err := c.fetchSeatHMACKey(requestCtx, token)
		ttl := seatDynamicKeyTTL
		if err != nil {
			// 动态配置异常时短暂缓存静态密钥，避免 token 校验反复等待配置接口。
			if strings.TrimSpace(c.Secret) == "" {
				return "", err
			}
			key = c.Secret
			ttl = seatFallbackKeyTTL
		}

		seatHMACCache.Lock()
		seatHMACCache.key = key
		seatHMACCache.until = time.Now().Add(ttl)
		seatHMACCache.Unlock()
		return key, nil
	})

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case result := <-resultCh:
		if result.Err != nil {
			return "", result.Err
		}
		key, ok := result.Val.(string)
		if !ok || key == "" {
			return "", errors.New("library: invalid hmac key result")
		}
		return key, nil
	}
}

func cachedSeatHMACKey() (string, bool) {
	seatHMACCache.RLock()
	defer seatHMACCache.RUnlock()
	if seatHMACCache.key == "" || !time.Now().Before(seatHMACCache.until) {
		return "", false
	}
	return seatHMACCache.key, true
}

func (c *Library) fetchSeatHMACKey(ctx context.Context, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", PG_URL_SEAT_SYSTEM_CONFIG, bytes.NewReader([]byte("{}")))
	if err != nil {
		return "", err
	}
	req.Header.Set("Token", token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("LoginType", "PC")
	resp, err := c.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := httpx.ReadResponse(resp)
	if err != nil {
		return "", err
	}
	envelope, err := decodeLibraryResponse(resp, body)
	if err != nil {
		return "", err
	}
	var data struct {
		HMACKey string `json:"hmacKey"`
	}
	if err := json.Unmarshal(envelope.Data, &data); err != nil || data.HMACKey == "" {
		return "", errors.New("library: system config did not contain hmacKey")
	}
	return decryptSeatHMACKey(data.HMACKey)
}

func decryptSeatHMACKey(encoded string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode hmacKey: %w", err)
	}
	block, err := aes.NewCipher([]byte("server_date_time"))
	if err != nil {
		return "", err
	}
	if len(ciphertext) == 0 || len(ciphertext)%aes.BlockSize != 0 {
		return "", errors.New("invalid AES-CBC ciphertext length")
	}
	plain := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, []byte("client_date_time")).CryptBlocks(plain, ciphertext)
	padding := int(plain[len(plain)-1])
	if padding < 1 || padding > aes.BlockSize || padding > len(plain) {
		return "", errors.New("invalid PKCS7 padding")
	}
	for _, value := range plain[len(plain)-padding:] {
		if int(value) != padding {
			return "", errors.New("invalid PKCS7 padding")
		}
	}
	return string(plain[:len(plain)-padding]), nil
}

// GetDiscussionAuthTokenFromLibrary 从图书馆系统中提取研讨室预约服务的 Token
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

	body, err := httpx.ReadResponse(resp)
	if err != nil {
		return "", err
	}

	respBody, err := decodeLibraryResponse(resp, body)
	if err != nil {
		return "", err
	}
	var data struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(respBody.Data, &data); err != nil || data.Token == "" {
		return "", errorx.Errorf("library: discussion token response has invalid data")
	}
	return data.Token, nil
}

// CheckLibraryDiscussionToken 验证研讨室预约服务Token的有效性
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
	body, err := httpx.ReadResponse(resp)
	if err != nil {
		return false, err
	}
	_, err = decodeLibraryResponse(resp, body)
	return err == nil, nil
}

func (c *Library) parseRawTokenFromUrl(urlStr string) (string, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}
	if token := parsedURL.Query().Get("token"); token != "" {
		return token, nil
	}
	fragment := parsedURL.Fragment
	if index := strings.Index(fragment, "?"); index >= 0 {
		fragment = fragment[index+1:]
	}
	params, err := url.ParseQuery(fragment)
	if err != nil {
		return "", errorx.Errorf("library: parse token fragment failed: %w", err)
	}
	token := params.Get("token")
	if token == "" {
		return "", errorx.New("library: project entry did not contain a token")
	}
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
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", errorx.Errorf("library: project entry returned HTTP %d", resp.StatusCode)
	}

	urlStr := resp.Request.URL.String()

	rawToken, err := c.parseRawTokenFromUrl(urlStr)
	if err != nil || rawToken == "" {
		return "", err
	}

	return rawToken, nil

}

func (c *Library) buildLibraryTokenUrl(prefix string) string {
	return fmt.Sprintf("%s?rand=%d", prefix, time.Now().UnixNano())
}

func decodeLibraryResponse(resp *http.Response, body []byte) (*response, error) {
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, errorx.Errorf("library: upstream returned HTTP %d", resp.StatusCode)
	}
	var result response
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, errorx.Errorf("library: decode upstream response failed: %w", err)
	}
	if !result.Status || (result.Code != 0 && result.Code != http.StatusOK) {
		return nil, errorx.Errorf("library: upstream rejected request, code:%d message:%s", result.Code, result.Message)
	}
	return &result, nil
}
