package crawler

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/asynccnu/ccnubox-be/be-library/tool"
	"github.com/asynccnu/ccnubox-be/common/pkg/metricsx"
	"github.com/google/uuid"
	"golang.org/x/sync/semaphore"
	"golang.org/x/sync/singleflight"
)

const (
	reminderTodayPath   = "/jsq/static/frontApi/user/lastMake"
	reminderHistoryPath = "/jsq/static/frontApi/user/history/%d/%d"
	reminderCurrentPath = "/jsq/static/frontApi/user/currentUseMake"
	reminderUserPath    = "/jsq/static/frontApi/user/getUserInfo"
	reminderDoorLogPath = "/jsq/static/frontApi/user/doorLog/%s"
	reminderSysSetPath  = "/jsq/static/public/cg/getSysSet/PC"
)

var ErrUpstreamStateUnknown = errors.New("library reminder upstream state unknown")

type ReminderCrawler interface {
	GetTodayReservations(context.Context, string) ([]ReminderReservation, error)
	GetRecentHistory(context.Context, string, HistoryWatermark) (HistoryPage, error)
	GetCurrentReservation(context.Context, string) (*ReminderReservation, error)
	GetUserState(context.Context, string) (LibraryUserState, error)
	GetDoorLogs(context.Context, string, string) ([]DoorLog, error)
}

type HistoryWatermark struct {
	ReservationID string
}

type HistoryPage struct {
	Reservations []ReminderReservation
	Complete     bool
}

type ReminderReservation struct {
	ID          string `json:"id"`
	SeatID      string `json:"seatId"`
	SeatLabel   string `json:"seatLabel"`
	MakeDateStr string `json:"makeDateStr"`
	MakeBegin   Minute `json:"makeBegin"`
	MakeEnd     Minute `json:"makeEnd"`
	Status      string `json:"status"`
	Location    string `json:"location"`
	Message     string `json:"message"`
	AwayRange   string `json:"awayRange"`
	AwayTimeM   int    `json:"awayTimeM"`
}

// UnmarshalJSON 同时兼容学校预约接口的新旧时间字段，以及数字或字符串状态。
func (r *ReminderReservation) UnmarshalJSON(raw []byte) error {
	var data struct {
		ID           string          `json:"id"`
		SeatID       string          `json:"seatId"`
		SeatLabel    string          `json:"seatLabel"`
		MakeDateStr  string          `json:"makeDateStr"`
		MakeBegin    json.RawMessage `json:"makeBegin"`
		MakeBeginStr json.RawMessage `json:"makeBeginStr"`
		MakeEnd      json.RawMessage `json:"makeEnd"`
		MakeEndStr   json.RawMessage `json:"makeEndStr"`
		Status       json.RawMessage `json:"status"`
		Location     string          `json:"location"`
		Message      string          `json:"message"`
		AwayRange    string          `json:"awayRange"`
		AwayTimeM    int             `json:"awayTimeM"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return err
	}
	begin, err := firstMinute(data.MakeBegin, data.MakeBeginStr)
	if err != nil {
		return fmt.Errorf("decode reservation begin: %w", err)
	}
	end, err := firstMinute(data.MakeEnd, data.MakeEndStr)
	if err != nil {
		return fmt.Errorf("decode reservation end: %w", err)
	}
	status, err := stringOrNumber(data.Status)
	if err != nil {
		return fmt.Errorf("decode reservation status: %w", err)
	}
	*r = ReminderReservation{
		ID: data.ID, SeatID: data.SeatID, SeatLabel: data.SeatLabel,
		MakeDateStr: data.MakeDateStr, MakeBegin: begin, MakeEnd: end,
		Status: status, Location: data.Location, Message: data.Message,
		AwayRange: data.AwayRange, AwayTimeM: data.AwayTimeM,
	}
	return nil
}

func (r ReminderReservation) Times() (time.Time, time.Time, error) {
	return r.timesIn(tool.GetLocation())
}

func (r ReminderReservation) timesIn(loc *time.Location) (time.Time, time.Time, error) {
	day, err := time.ParseInLocation("2006-01-02", r.MakeDateStr, loc)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("parse reservation date: %w", err)
	}
	if int(r.MakeBegin) < 0 || int(r.MakeBegin) > 24*60 || int(r.MakeEnd) < 0 || int(r.MakeEnd) > 24*60 || r.MakeEnd <= r.MakeBegin {
		return time.Time{}, time.Time{}, errors.New("invalid reservation minute range")
	}
	return day.Add(time.Duration(r.MakeBegin) * time.Minute), day.Add(time.Duration(r.MakeEnd) * time.Minute), nil
}

type Minute int

func (m *Minute) UnmarshalJSON(raw []byte) error {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return errors.New("minute is missing")
	}
	var number int
	if err := json.Unmarshal(raw, &number); err == nil {
		*m = Minute(number)
		return nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return errors.New("minute must be a number or HH:mm string")
	}
	if value, err := strconv.Atoi(text); err == nil {
		*m = Minute(value)
		return nil
	}
	parsed, err := time.Parse("15:04", text)
	if err != nil {
		return fmt.Errorf("invalid minute %q", text)
	}
	*m = Minute(parsed.Hour()*60 + parsed.Minute())
	return nil
}

func firstMinute(values ...json.RawMessage) (Minute, error) {
	for _, raw := range values {
		trimmed := bytes.TrimSpace(raw)
		if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
			continue
		}
		var text string
		if err := json.Unmarshal(trimmed, &text); err == nil && strings.TrimSpace(text) == "" {
			continue
		}
		var minute Minute
		if err := minute.UnmarshalJSON(trimmed); err != nil {
			return 0, err
		}
		return minute, nil
	}
	return 0, errors.New("minute is missing")
}

func stringOrNumber(raw json.RawMessage) (string, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return "", nil
	}
	var text string
	if err := json.Unmarshal(trimmed, &text); err == nil {
		return text, nil
	}
	var number json.Number
	if err := json.Unmarshal(trimmed, &number); err == nil && number != "" {
		return number.String(), nil
	}
	return "", errors.New("value must be a string or number")
}

type LibraryUserState struct {
	BreachNum     int    `json:"breachNum"`
	ScoreNum      int    `json:"scoreNum"`
	BlackTime     string `json:"blackTime"`
	BlackMessage  string `json:"blackMessage"`
	CycleTimeName string `json:"cycleTimeName"`
}

type DoorLog json.RawMessage

type reminderEnvelope struct {
	Status  bool            `json:"status"`
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type upstreamError struct {
	Endpoint string
	HTTPCode int
	Code     int
	Message  string
	Cause    error
}

func (e *upstreamError) Error() string {
	if e.Cause == nil {
		return fmt.Sprintf("%v: endpoint=%s http=%d code=%d message=%s", ErrUpstreamStateUnknown, e.Endpoint, e.HTTPCode, e.Code, e.Message)
	}
	return fmt.Sprintf("%v: endpoint=%s http=%d code=%d message=%s cause=%v", ErrUpstreamStateUnknown, e.Endpoint, e.HTTPCode, e.Code, e.Message, e.Cause)
}
func (e *upstreamError) Unwrap() error { return e.Cause }
func (e *upstreamError) Is(target error) bool {
	return target == ErrUpstreamStateUnknown || errors.Is(e.Cause, target)
}

type ReminderHTTPClient struct {
	client          *http.Client
	baseURL         string
	requestTimeout  time.Duration
	historyPageSize int
	historyMaxPages int
	historyLookback int
	requestSpacing  time.Duration
	rateGate        *semaphore.Weighted
	nextRequest     time.Time

	keyMu          sync.Mutex
	keyGroup       singleflight.Group
	hmacKey        string
	hmacKeyUntil   time.Time
	hmacKeyVersion uint64
	metrics        *metricsx.LibraryReminderMetrics
}

type hmacKeySnapshot struct {
	key     string
	version uint64
}

func NewReminderCrawler(client *http.Client, requestTimeout time.Duration, historyPageSize, lookbackDays, upstreamQPS int, metricSet ...*metricsx.LibraryReminderMetrics) *ReminderHTTPClient {
	if requestTimeout <= 0 {
		requestTimeout = 8 * time.Second
	}
	if historyPageSize <= 0 {
		historyPageSize = 20
	}
	if lookbackDays <= 0 {
		lookbackDays = 3
	}
	if upstreamQPS <= 0 {
		upstreamQPS = 20
	}
	result := &ReminderHTTPClient{client: client, baseURL: BaseDomain, requestTimeout: requestTimeout, historyPageSize: historyPageSize, historyMaxPages: 100, historyLookback: lookbackDays, requestSpacing: time.Second / time.Duration(upstreamQPS), rateGate: semaphore.NewWeighted(1)}
	if len(metricSet) > 0 {
		result.metrics = metricSet[0]
	}
	return result
}

func (c *ReminderHTTPClient) GetTodayReservations(ctx context.Context, token string) ([]ReminderReservation, error) {
	raw, err := c.request(ctx, token, reminderTodayPath)
	if err != nil {
		return nil, err
	}
	var rows []ReminderReservation
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, fmt.Errorf("%w: decode today reservations", ErrUpstreamStateUnknown)
	}
	if err := validateReservations(rows); err != nil {
		return nil, err
	}
	return rows, nil
}

func (c *ReminderHTTPClient) GetRecentHistory(ctx context.Context, token string, watermark HistoryWatermark) (HistoryPage, error) {
	result := HistoryPage{Complete: true}
	loc := tool.GetLocation()
	cutoff := time.Now().In(loc).AddDate(0, 0, -c.historyLookback).Format("2006-01-02")
	for page := 0; page < c.historyMaxPages; page++ {
		raw, err := c.request(ctx, token, fmt.Sprintf(reminderHistoryPath, page, c.historyPageSize))
		if err != nil {
			return HistoryPage{}, err
		}
		var data struct {
			Count int                   `json:"count"`
			List  []ReminderReservation `json:"list"`
		}
		if err := json.Unmarshal(raw, &data); err != nil {
			return HistoryPage{}, fmt.Errorf("%w: decode reservation history", ErrUpstreamStateUnknown)
		}
		if err := validateReservations(data.List); err != nil {
			return HistoryPage{}, err
		}
		for _, row := range data.List {
			if watermark.ReservationID != "" && row.ID == watermark.ReservationID {
				return result, nil
			}
			if row.MakeDateStr < cutoff {
				return result, nil
			}
			result.Reservations = append(result.Reservations, row)
		}
		if len(data.List) < c.historyPageSize || len(result.Reservations) >= data.Count {
			return result, nil
		}
	}
	result.Complete = false
	return result, nil
}

func (c *ReminderHTTPClient) GetCurrentReservation(ctx context.Context, token string) (*ReminderReservation, error) {
	raw, err := c.request(ctx, token, reminderCurrentPath)
	if err != nil {
		return nil, err
	}
	data := bytes.TrimSpace(raw)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) || bytes.Equal(data, []byte(`""`)) {
		return nil, nil
	}
	if len(data) > 0 && data[0] == '[' {
		return nil, fmt.Errorf("%w: current reservation data was an array", ErrUpstreamStateUnknown)
	}
	var row ReminderReservation
	if err := json.Unmarshal(data, &row); err != nil {
		return nil, fmt.Errorf("%w: decode current reservation", ErrUpstreamStateUnknown)
	}
	if err := validateReservations([]ReminderReservation{row}); err != nil {
		return nil, err
	}
	return &row, nil
}

func (c *ReminderHTTPClient) GetUserState(ctx context.Context, token string) (LibraryUserState, error) {
	raw, err := c.request(ctx, token, reminderUserPath)
	if err != nil {
		return LibraryUserState{}, err
	}
	data := bytes.TrimSpace(raw)
	if len(data) == 0 || data[0] != '{' {
		return LibraryUserState{}, fmt.Errorf("%w: user state data must be an object", ErrUpstreamStateUnknown)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return LibraryUserState{}, fmt.Errorf("%w: decode user state", ErrUpstreamStateUnknown)
	}
	for _, name := range []string{"breachNum", "scoreNum", "blackTime", "blackMessage", "cycleTimeName"} {
		if _, exists := fields[name]; !exists {
			return LibraryUserState{}, fmt.Errorf("%w: user state is missing %s", ErrUpstreamStateUnknown, name)
		}
	}
	for _, name := range []string{"breachNum", "scoreNum"} {
		if bytes.Equal(bytes.TrimSpace(fields[name]), []byte("null")) {
			return LibraryUserState{}, fmt.Errorf("%w: user state field %s is null", ErrUpstreamStateUnknown, name)
		}
	}
	var state LibraryUserState
	if err := json.Unmarshal(data, &state); err != nil {
		return LibraryUserState{}, fmt.Errorf("%w: decode user state", ErrUpstreamStateUnknown)
	}
	if len(state.CycleTimeName) > 128 || len(state.BlackTime) > 255 || len(state.BlackMessage) > 60<<10 {
		return LibraryUserState{}, fmt.Errorf("%w: user state fields exceed storage limits", ErrUpstreamStateUnknown)
	}
	return state, nil
}

func (c *ReminderHTTPClient) GetDoorLogs(ctx context.Context, token, date string) ([]DoorLog, error) {
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return nil, errors.New("door log date must use YYYY-MM-DD")
	}
	raw, err := c.request(ctx, token, fmt.Sprintf(reminderDoorLogPath, date))
	if err != nil {
		return nil, err
	}
	var result []DoorLog
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("%w: decode door logs", ErrUpstreamStateUnknown)
	}
	return result, nil
}

func validateReservations(rows []ReminderReservation) error {
	loc := tool.GetLocation()
	for _, row := range rows {
		if strings.TrimSpace(row.ID) == "" || strings.TrimSpace(row.MakeDateStr) == "" {
			return fmt.Errorf("%w: reservation is missing identity or date", ErrUpstreamStateUnknown)
		}
		if len(row.ID) > 128 || len(row.SeatID) > 128 || len(row.SeatLabel) > 128 || len(row.Location) > 512 || len(row.Status) > 128 || len(row.MakeDateStr) != len("2006-01-02") {
			return fmt.Errorf("%w: reservation fields exceed storage limits", ErrUpstreamStateUnknown)
		}
		if _, _, err := row.timesIn(loc); err != nil {
			return fmt.Errorf("%w: invalid reservation %s: %v", ErrUpstreamStateUnknown, row.ID, err)
		}
	}
	return nil
}

func (c *ReminderHTTPClient) request(parent context.Context, token, path string) (json.RawMessage, error) {
	if token == "" {
		return nil, fmt.Errorf("%w: empty token", ErrUpstreamStateUnknown)
	}
	key, version, err := c.signingKey(parent, token, false, 0)
	if err != nil {
		return nil, err
	}
	data, err := c.signedRequest(parent, token, key, path)
	if err == nil {
		return data, nil
	}
	// 仅在明确的 401/403 认证拒绝时刷新 HMAC 重放一次；
	// 超时、5xx、业务错误直接上抛，避免外层重试放大上游故障流量。
	if !isAuthRejection(err) {
		return nil, err
	}
	key, _, refreshErr := c.signingKey(parent, token, true, version)
	if refreshErr != nil {
		return nil, err
	}
	return c.signedRequest(parent, token, key, path)
}

// isAuthRejection 判断是否为明确的认证/签名拒绝（HTTP 或业务码 401/403）。
func isAuthRejection(err error) bool {
	var upstream *upstreamError
	if !errors.As(err, &upstream) {
		return false
	}
	return upstream.HTTPCode == http.StatusUnauthorized || upstream.HTTPCode == http.StatusForbidden ||
		upstream.Code == http.StatusUnauthorized || upstream.Code == http.StatusForbidden
}

func (c *ReminderHTTPClient) signedRequest(parent context.Context, token, key, path string) (json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(parent, c.requestTimeout)
	defer cancel()
	now := time.Now().UnixMilli()
	requestID := uuid.NewString()
	message := fmt.Sprintf("seat::%s::%d::POST", requestID, now)
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte(message))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader([]byte("{}")))
	if err != nil {
		return nil, fmt.Errorf("%w: create request", ErrUpstreamStateUnknown)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("token", token)
	req.Header.Set("loginType", "PC")
	req.Header.Set("X-request-id", requestID)
	req.Header.Set("X-request-date", strconv.FormatInt(now, 10))
	req.Header.Set("X-hmac-request-key", hex.EncodeToString(mac.Sum(nil)))
	return c.do(req)
}

func (c *ReminderHTTPClient) signingKey(parent context.Context, token string, force bool, observedVersion uint64) (string, uint64, error) {
	if cached, ok := c.cachedSigningKey(force, observedVersion); ok {
		return cached.key, cached.version, nil
	}

	result := c.keyGroup.DoChan("hmac", func() (any, error) {
		// 请求进入 singleflight 前缓存可能已由另一个刷新者更新。
		if cached, ok := c.cachedSigningKey(force, observedVersion); ok {
			return cached, nil
		}
		key, err := c.fetchSigningKey(context.WithoutCancel(parent), token)
		if err != nil {
			return hmacKeySnapshot{}, err
		}
		c.keyMu.Lock()
		defer c.keyMu.Unlock()
		c.hmacKey = key
		c.hmacKeyUntil = time.Now().Add(10 * time.Minute)
		c.hmacKeyVersion++
		return hmacKeySnapshot{key: key, version: c.hmacKeyVersion}, nil
	})
	select {
	case <-parent.Done():
		return "", 0, parent.Err()
	case shared := <-result:
		if shared.Err != nil {
			return "", 0, shared.Err
		}
		key := shared.Val.(hmacKeySnapshot)
		return key.key, key.version, nil
	}
}

func (c *ReminderHTTPClient) cachedSigningKey(force bool, observedVersion uint64) (hmacKeySnapshot, bool) {
	c.keyMu.Lock()
	defer c.keyMu.Unlock()
	if c.hmacKey == "" || !time.Now().Before(c.hmacKeyUntil) {
		return hmacKeySnapshot{}, false
	}
	if force && c.hmacKeyVersion == observedVersion {
		return hmacKeySnapshot{}, false
	}
	return hmacKeySnapshot{key: c.hmacKey, version: c.hmacKeyVersion}, true
}

func (c *ReminderHTTPClient) fetchSigningKey(parent context.Context, token string) (string, error) {
	ctx, cancel := context.WithTimeout(parent, c.requestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+reminderSysSetPath, bytes.NewReader([]byte("{}")))
	if err != nil {
		return "", fmt.Errorf("%w: create system config request", ErrUpstreamStateUnknown)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("token", token)
	req.Header.Set("loginType", "PC")
	raw, err := c.do(req)
	if err != nil {
		return "", err
	}
	var data struct {
		HMACKey string `json:"hmacKey"`
	}
	if err := json.Unmarshal(raw, &data); err != nil || data.HMACKey == "" {
		return "", fmt.Errorf("%w: invalid system hmac config", ErrUpstreamStateUnknown)
	}
	key, err := decryptHMACKey(data.HMACKey)
	if err != nil || key == "" {
		return "", fmt.Errorf("%w: decrypt system hmac config", ErrUpstreamStateUnknown)
	}
	return key, nil
}

func (c *ReminderHTTPClient) do(req *http.Request) (data json.RawMessage, err error) {
	started := time.Now()
	endpoint := reminderMetricEndpoint(req.URL.Path)
	if c.metrics != nil {
		defer func() {
			result := "success"
			if err != nil {
				result = ClassifyUpstreamError(err)
			}
			c.metrics.UpstreamRequestsTotal.WithLabelValues(endpoint, result).Inc()
			c.metrics.UpstreamDurationSeconds.WithLabelValues(endpoint).Observe(time.Since(started).Seconds())
		}()
	}
	if err := c.waitRate(req.Context()); err != nil {
		return nil, &upstreamError{Endpoint: endpoint, Cause: err}
	}
	resp, err := c.client.Do(req)
	if err != nil {
		// http.Client 失败时返回 *url.Error，其文本包含完整请求 URL；
		// 仅保留底层错误作为 cause，避免完整 URL 进入日志与任务错误样例。
		var urlErr *url.Error
		if errors.As(err, &urlErr) && urlErr.Err != nil {
			err = urlErr.Err
		}
		return nil, &upstreamError{Endpoint: endpoint, Cause: err}
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, &upstreamError{Endpoint: endpoint, HTTPCode: resp.StatusCode, Cause: err}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &upstreamError{Endpoint: endpoint, HTTPCode: resp.StatusCode}
	}
	var env reminderEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, &upstreamError{Endpoint: endpoint, HTTPCode: resp.StatusCode, Cause: err}
	}
	if !env.Status || (env.Code != 0 && env.Code != http.StatusOK) {
		return nil, &upstreamError{Endpoint: endpoint, HTTPCode: resp.StatusCode, Code: env.Code, Message: env.Message}
	}
	if env.Data == nil {
		return nil, &upstreamError{Endpoint: endpoint, HTTPCode: resp.StatusCode, Code: env.Code, Message: "missing data"}
	}
	return env.Data, nil
}

func reminderMetricEndpoint(path string) string {
	switch {
	case path == reminderTodayPath:
		return "last_make"
	case strings.HasPrefix(path, "/jsq/static/frontApi/user/history/"):
		return "history"
	case path == reminderCurrentPath:
		return "current_use_make"
	case path == reminderUserPath:
		return "user_info"
	case strings.HasPrefix(path, "/jsq/static/frontApi/user/doorLog/"):
		return "door_log"
	case path == reminderSysSetPath:
		return "system_config"
	default:
		return "unknown"
	}
}

// ClassifyUpstreamError 将上游请求错误归类为低基数类别，供指标与 service 层批量错误分类复用。
// 返回值为 timeout/auth_error/http_error/business_error/network_or_decode_error/invalid_response。
func ClassifyUpstreamError(err error) string {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	var upstream *upstreamError
	if errors.As(err, &upstream) {
		if isAuthRejection(err) {
			return "auth_error"
		}
		if upstream.HTTPCode != 0 && (upstream.HTTPCode < 200 || upstream.HTTPCode >= 300) {
			return "http_error"
		}
		if upstream.Code != 0 && upstream.Code != http.StatusOK {
			return "business_error"
		}
		if upstream.Cause != nil {
			return "network_or_decode_error"
		}
	}
	return "invalid_response"
}

func (c *ReminderHTTPClient) waitRate(ctx context.Context) error {
	// 仅限制请求发起速率，不限制在途并发；放行后即释放闸门，不等待 HTTP 请求完成。
	// 只由队首等待下一次放行，不为排队请求预占未来时隙；取消可直接退出队列。
	if err := c.rateGate.Acquire(ctx, 1); err != nil {
		return err
	}
	defer c.rateGate.Release(1)
	if delay := time.Until(c.nextRequest); delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	c.nextRequest = time.Now().Add(c.requestSpacing)
	return nil
}

func decryptHMACKey(ciphertextBase64 string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextBase64)
	if err != nil {
		return "", err
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
