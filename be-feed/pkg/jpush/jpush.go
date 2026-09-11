package jpush

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Scorpio69t/jpush-api-golang-client"
	"github.com/mitchellh/mapstructure"
)

const (
	jpushHTTPTimeout      = 15 * time.Second
	jpushResponseMaxBytes = 1 << 20
	jpushErrorMaxBytes    = 1024
)

type client struct {
	pf           *jpush.Platform
	o            *jpush.Options
	appKey       string
	masterSecret string
	httpClient   *http.Client
	pushURL      string
	cidURL       string
}

type PushClient interface {
	GetCID(context.Context) (string, error)
	Push(context.Context, []string, PushData) error
}

type PushData struct {
	ContentType string            `json:"content_type"`
	Extras      map[string]string `json:"extras"`
	MsgContent  string            `json:"msg_content"`
	Title       string            `json:"title"`
	Cid         string            `json:"cid,omitempty"`
}

type JPushConfig struct {
	AppKey       string `json:"app_key"`
	MasterSecret string `json:"master_secret"`
	HUAWEI       struct {
		Category string `json:"category"`
	}
	XIAOMI struct {
		ChannelId string `json:"channel_id"`
	}
	OPPO struct {
		ChannelId string `json:"channel_id"`
	}
}

func NewJPushClient(cfg *JPushConfig) PushClient {
	//极光推送客户端
	var pf jpush.Platform
	//设定为推送给所有平台
	pf.All()
	//配置极光推送选项
	var o jpush.Options
	o.SetApnsProduction(true)
	o.AddThirdPartyChannel(jpush.XIAOMI, jpush.ThirdPartyOptions{ChannelId: cfg.XIAOMI.ChannelId})
	o.AddThirdPartyChannel(jpush.HUAWEI, jpush.ThirdPartyOptions{Category: cfg.HUAWEI.Category})
	o.AddThirdPartyChannel(jpush.OPPO, jpush.ThirdPartyOptions{ChannelId: cfg.OPPO.ChannelId})

	return &client{
		pf:           &pf,
		o:            &o,
		appKey:       cfg.AppKey,
		masterSecret: cfg.MasterSecret,
		httpClient:   &http.Client{Timeout: jpushHTTPTimeout},
		pushURL:      jpush.HOST_PUSH,
		cidURL:       jpush.HOST_CID,
	}
}

func (c *client) GetCID(ctx context.Context) (string, error) {
	u, err := url.Parse(c.cidURL)
	if err != nil {
		return "", fmt.Errorf("jpush: parse cid url: %w", err)
	}
	query := u.Query()
	query.Set("count", "1")
	query.Set("type", "push")
	u.RawQuery = query.Encode()

	data, err := c.do(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	var response struct {
		CIDList []string `json:"cidlist"`
	}
	if err = json.Unmarshal(data, &response); err != nil {
		return "", fmt.Errorf("jpush: decode cid response: %w", err)
	}
	if len(response.CIDList) == 0 || strings.TrimSpace(response.CIDList[0]) == "" {
		return "", errors.New("jpush: get cid returned empty list")
	}
	return response.CIDList[0], nil
}

func (c *client) Push(ctx context.Context, ids []string, pushData PushData) error {
	// 如果无推送目标直接跳过
	if len(ids) == 0 {
		return nil
	}

	//设置推送对象
	var at jpush.Audience

	at.SetID(ids)

	// 设置智能推送以及智能推送的内容
	var n jpush.Notification

	var extras map[string]interface{}
	extras = make(map[string]interface{}, len(pushData.Extras))
	for k, v := range pushData.Extras {
		extras[k] = v
	}

	//推送给所有的平台,包括安卓,ios,windows
	n.SetAndroid(&jpush.AndroidNotification{
		Alert:       pushData.MsgContent,
		AlertType:   7,
		BadgeAddNum: 1, //每次提醒增加的角标数量
		BuilderID:   1,
		Style:       0, //样式字段
		Title:       pushData.Title,
		Priority:    1,
		Extras:      extras,
	})

	n.SetIos(&jpush.IosNotification{
		Alert:             pushData.MsgContent,
		Badge:             1,
		ContentAvailable:  false,
		InterruptionLevel: "active",
		MutableContent:    true,
	})

	//加载推送
	payload := jpush.NewPayLoad()
	payload.Cid = pushData.Cid
	payload.SetOptions(c.o)
	payload.SetPlatform(c.pf)
	payload.SetAudience(&at)
	payload.SetNotification(&n)
	var interfaceMap map[string]interface{}

	// 使用 解码成map[string]interface{}
	err := mapstructure.Decode(pushData.Extras, &interfaceMap)
	if err != nil {
		return err
	}

	//将发送的消息改成byte类型
	data, err := payload.Bytes()
	if err != nil {
		return err
	}

	// 依赖库的 Push 使用无超时 http.Client，需在此处直接发送以传递 worker context。
	responseBody, err := c.do(ctx, http.MethodPost, c.pushURL, data)
	if err != nil {
		return err
	}
	var response struct {
		MsgID json.RawMessage `json:"msg_id"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return fmt.Errorf("jpush: decode push response: %w", err)
	}
	if !validMsgID(response.MsgID) {
		return fmt.Errorf("jpush: push rejected: %s", boundedResponse(responseBody))
	}
	return nil
}

// validMsgID 严格校验 msg_id 必须是非空字符串或合法数值，
// 拒绝缺失、null、空串、对象等异常 200 响应，防止投递被误标记为已发送。
func validMsgID(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return false
	}
	var text string
	if err := json.Unmarshal(trimmed, &text); err == nil {
		return strings.TrimSpace(text) != ""
	}
	var number json.Number
	return json.Unmarshal(trimmed, &number) == nil
}

func (c *client) do(ctx context.Context, method, endpoint string, body []byte) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return nil, fmt.Errorf("jpush: create request: %w", err)
	}
	req.Header.Set("Charset", "UTF-8")
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.appKey, c.masterSecret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jpush: send request: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, jpushResponseMaxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("jpush: read response: %w", err)
	}
	if len(data) > jpushResponseMaxBytes {
		return nil, errors.New("jpush: response exceeds size limit")
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("jpush: HTTP %d: %s", resp.StatusCode, boundedResponse(data))
	}
	return data, nil
}

func boundedResponse(data []byte) string {
	// 先替换非法 UTF-8 字节再沿字符边界截断；按字节硬截断会把多字节字符切开，
	// 产生的非法序列写入 utf8mb4 字段时会被 MySQL 拒绝。
	text := strings.TrimSpace(strings.ToValidUTF8(string(data), string(utf8.RuneError)))
	return truncateUTF8(text, jpushErrorMaxBytes)
}

// truncateUTF8 在不超过 limit 字节的前提下沿 UTF-8 字符边界截断。
func truncateUTF8(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	for limit > 0 && !utf8.RuneStart(text[limit]) {
		limit--
	}
	return text[:limit]
}
