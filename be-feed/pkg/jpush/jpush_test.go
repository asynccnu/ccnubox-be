package jpush

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestPushHonorsContext(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 读完请求体，服务端才会开始探测客户端断开并取消 r.Context()
		_, _ = io.Copy(io.Discard, r.Body)
		close(started)
		<-r.Context().Done()
	}))
	defer server.Close()

	client := NewJPushClient(&JPushConfig{}).(*client)
	client.pushURL = server.URL
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := client.Push(ctx, []string{"token"}, PushData{Title: "标题", MsgContent: "内容"})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Push() error = %v, want context deadline exceeded", err)
	}
	select {
	case <-started:
	default:
		t.Fatal("JPush request was not sent")
	}
}

func TestPushAcceptsJPushResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, _, ok := r.BasicAuth(); !ok {
			t.Error("request did not contain basic auth")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sendno":"0","msg_id":"1"}`))
	}))
	defer server.Close()

	client := NewJPushClient(&JPushConfig{}).(*client)
	client.pushURL = server.URL
	if err := client.Push(context.Background(), []string{"token"}, PushData{Title: "标题", MsgContent: "内容"}); err != nil {
		t.Fatalf("Push() error = %v", err)
	}
}

func TestPushAcceptsNumericMsgID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sendno":"0","msg_id":123456789}`))
	}))
	defer server.Close()

	client := NewJPushClient(&JPushConfig{}).(*client)
	client.pushURL = server.URL
	if err := client.Push(context.Background(), []string{"token"}, PushData{Title: "标题", MsgContent: "内容"}); err != nil {
		t.Fatalf("Push() error = %v", err)
	}
}

// TestPushRejectsAbnormalMsgID 覆盖带异常 msg_id 的 200 响应，防止投递被误标记为已发送。
func TestPushRejectsAbnormalMsgID(t *testing.T) {
	abnormal := []string{
		`{}`,
		`{"sendno":"0"}`,
		`{"msg_id":null}`,
		`{"msg_id":""}`,
		`{"msg_id":"   "}`,
		`{"msg_id":{}}`,
		`{"msg_id":[]}`,
		`{"msg_id":true}`,
	}
	for _, body := range abnormal {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(body))
		}))
		client := NewJPushClient(&JPushConfig{}).(*client)
		client.pushURL = server.URL
		if err := client.Push(context.Background(), []string{"token"}, PushData{Title: "标题", MsgContent: "内容"}); err == nil {
			t.Errorf("Push() with body %s succeeded, want rejection", body)
		}
		server.Close()
	}
}

// TestBoundedResponseKeepsValidUTF8 验证含非法字节与中文的响应破截断后仍是合法 UTF-8，
// 避免写入 utf8mb4 字段时被 MySQL 拒绝。
func TestBoundedResponseKeepsValidUTF8(t *testing.T) {
	// 构造超过 1024 字节且中文对齐在截断点中间的响应，再混入非法 UTF-8 字节。
	body := []byte(strings.Repeat("\u4e2d\u6587", 256)) // 2048 字节
	body = append(body, 0xFF, 0xC0)                     // 非法字节
	text := boundedResponse(body)
	if len(text) > jpushErrorMaxBytes {
		t.Fatalf("boundedResponse() length = %d, want <= %d", len(text), jpushErrorMaxBytes)
	}
	if !utf8.ValidString(text) {
		t.Fatalf("boundedResponse() produced invalid UTF-8: %q", text)
	}
}
