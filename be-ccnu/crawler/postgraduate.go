package crawler

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"

	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
)

const (
	postgraduateURL      = "https://grd.ccnu.edu.cn"
	publicKeyURL         = postgraduateURL + "/yjsxt/xtgl/login_getPublicKey.html"
	loginPostgraduateURL = postgraduateURL + "/yjsxt/xtgl/login_slogin.html"
)

type PostGraduate struct {
	client *http.Client
}

func NewPostGraduate(client *http.Client) *PostGraduate {
	return &PostGraduate{client: client}
}

// 1. 获取 RSA 公钥
func (c *PostGraduate) FetchPublicKey(ctx context.Context) (*rsa.PublicKey, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", publicKeyURL, nil)
	if err != nil {
		return nil, errorx.Errorf("postgraduate: create public key request failed: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", postgraduateURL+"/yjsxt/")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, errorx.Errorf("postgraduate: send public key request failed: %w", err)
	}
	defer resp.Body.Close()

	var data rsaPublicKeyResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, errorx.Errorf("postgraduate: decode public key response failed: %w", err)
	}

	pubKey, err := parseRSAPublicKey(data.Modulus, data.Exponent)
	if err != nil {
		return nil, errorx.Errorf("postgraduate: parse public key failed: %w", err)
	}

	return pubKey, nil
}

// 2. 登录研究生系统
func (c *PostGraduate) LoginPostgraduateSystem(
	ctx context.Context,
	username,
	password string,
	pubKey *rsa.PublicKey,
) error {

	encPwd, err := encryptPasswordJSStyle(password, pubKey)
	if err != nil {
		return errorx.Errorf("postgraduate: encrypt password failed: %w", err)
	}

	form := url.Values{}
	form.Set("csrftoken", "")
	form.Set("yhm", username)
	form.Set("mm", encPwd)

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		loginPostgraduateURL,
		bytes.NewBufferString(form.Encode()),
	)
	if err != nil {
		return errorx.Errorf("postgraduate: create login request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", postgraduateURL+"/yjsxt/")
	req.Header.Set("Origin", postgraduateURL)
	req.Header.Set("Host", "grd.ccnu.edu.cn")

	resp, err := c.client.Do(req)
	if err != nil {
		return errorx.Errorf("postgraduate: send login request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errorx.Errorf("postgraduate: read login response failed: %w", err)
	}

	if strings.Contains(string(body), "用户名或密码不正确") {
		return INCorrectPASSWORD
	}

	return nil
}

// 3. 登录并获取 Cookie
func (c *PostGraduate) GetCookie(
	ctx context.Context,
	stuId,
	password string,
	pubKey *rsa.PublicKey,
) (string, error) {

	encPwd, err := encryptPasswordJSStyle(password, pubKey)
	if err != nil {
		return "", errorx.Errorf("postgraduate: encrypt password failed: %w", err)
	}

	form := url.Values{}
	form.Set("csrftoken", "")
	form.Set("yhm", stuId)
	form.Set("mm", encPwd)

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		loginPostgraduateURL,
		bytes.NewBufferString(form.Encode()),
	)
	if err != nil {
		return "", errorx.Errorf("postgraduate: create cookie request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", postgraduateURL+"/yjsxt/")
	req.Header.Set("Origin", postgraduateURL)
	req.Header.Set("Host", "grd.ccnu.edu.cn")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", errorx.Errorf("postgraduate: send cookie request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errorx.Errorf("postgraduate: unexpected status code %d", resp.StatusCode)
	}

	var JSESSIONID, route string
	rootURL, _ := url.Parse("https://grd.ccnu.edu.cn/yjsxt")

	for _, cookie := range c.client.Jar.Cookies(rootURL) {
		switch cookie.Name {
		case "JSESSIONID":
			JSESSIONID = cookie.Value
		case "route":
			route = cookie.Value
		}
	}

	if JSESSIONID == "" || route == "" {
		return "", errorx.Errorf("postgraduate: required cookie missing")
	}

	return fmt.Sprintf("JSESSIONID=%s;route=%s", JSESSIONID, route), nil
}

type rsaPublicKeyResponse struct {
	Modulus  string `json:"modulus"`
	Exponent string `json:"exponent"`
}

func parseRSAPublicKey(modBase64, expBase64 string) (*rsa.PublicKey, error) {
	modBytes, err := base64.StdEncoding.DecodeString(modBase64)
	if err != nil {
		return nil, errorx.Errorf("rsa: decode modulus failed: %w", err)
	}

	expBytes, err := base64.StdEncoding.DecodeString(expBase64)
	if err != nil {
		return nil, errorx.Errorf("rsa: decode exponent failed: %w", err)
	}

	modulus := new(big.Int).SetBytes(modBytes)
	exponent := new(big.Int).SetBytes(expBytes)

	return &rsa.PublicKey{
		N: modulus,
		E: int(exponent.Int64()),
	}, nil
}

func encryptPasswordJSStyle(password string, pubKey *rsa.PublicKey) (string, error) {
	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, pubKey, []byte(password))
	if err != nil {
		return "", errorx.Errorf("rsa: encrypt password failed: %w", err)
	}

	hexStr := hex.EncodeToString(encrypted)
	hexBytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return "", errorx.Errorf("rsa: hex decode failed: %w", err)
	}

	return base64.StdEncoding.EncodeToString(hexBytes), nil
}
