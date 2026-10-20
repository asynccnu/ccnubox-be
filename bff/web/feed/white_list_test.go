package feed

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/alicebob/miniredis/v2"
	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/service/activity"
	service "github.com/asynccnu/ccnubox-be/bff/service/feed"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	"github.com/asynccnu/ccnubox-be/bff/web/middleware"
	feedv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feed/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger/zapx"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type officialClientStub struct {
	feedClientStub
	selected   *feedv1.SetOfficialWhiteListReq
	published  *feedv1.PublicMuxiOfficialMSGReq
	publishErr error
}

func (s *officialClientStub) SetOfficialWhiteList(_ context.Context, r *feedv1.SetOfficialWhiteListReq, _ ...grpc.CallOption) (*feedv1.SetOfficialWhiteListResp, error) {
	s.selected = r
	return &feedv1.SetOfficialWhiteListResp{Strategy: r.Strategy, UserCount: int32(len(r.StudentIds)), ExpiresAt: r.ExpiresAt}, nil
}
func (s *officialClientStub) PublicMuxiOfficialMSG(_ context.Context, r *feedv1.PublicMuxiOfficialMSGReq, _ ...grpc.CallOption) (*feedv1.PublicMuxiOfficialMSGResp, error) {
	s.published = r
	return &feedv1.PublicMuxiOfficialMSGResp{}, s.publishErr
}
func officialRouter(f *officialClientStub, clients ...redis.Cmdable) *gin.Engine {
	var client redis.Cmdable
	if len(clients) > 0 {
		client = clients[0]
	}
	gin.SetMode(gin.TestMode)
	g := gin.New()
	g.Use(middleware.NewLoggerMiddleware(zapx.NewZapLogger(zap.NewNop())).MiddlewareFunc())
	h := NewFeedHandler(f, map[string]struct{}{"admin": {}}, service.NewWhiteListService(client, f, service.WhiteListOptions{}))
	auth := func(c *gin.Context) { ginx.SetClaims(c, ijwt.UserClaims{StudentId: c.GetHeader("X-Test-User")}) }
	h.RegisterRoutes(g.Group("/api/v1"), auth, auth)
	return g
}
func officialRequest(g *gin.Engine, path, body, user string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/feed/"+path, bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Test-User", user)
	w := httptest.NewRecorder()
	g.ServeHTTP(w, r)
	return w
}
func TestWhiteListHTTPOnlyAcceptsStrategy(t *testing.T) {
	f := &officialClientStub{}
	g := officialRouter(f)
	w := officialRequest(g, "setWhiteList", `{"strategy":"all"}`, "admin")
	if w.Code != 200 || f.selected == nil || !f.selected.All || len(f.selected.StudentIds) != 0 {
		t.Fatalf("response=%d %s selected=%+v", w.Code, w.Body.String(), f.selected)
	}
	for _, body := range []string{`{"strategy":"script","student_ids":["a"]}`, `{"strategy":"script","command":"anything"}`, `{} {}`, `{"strategy":0}`} {
		f.selected = nil
		w = officialRequest(g, "setWhiteList", body, "admin")
		if w.Code != 422 || f.selected != nil {
			t.Fatalf("invalid input: %d %s", w.Code, w.Body.String())
		}
	}
	w = officialRequest(g, "setWhiteList", `{invalid`, "user")
	if w.Code != 403 {
		t.Fatalf("admin check: %d %s", w.Code, w.Body.String())
	}
	w = officialRequest(g, "setWhiteList", `{"strategy":"unknown"}`, "admin")
	if w.Code != 400 {
		t.Fatalf("strategy: %d %s", w.Code, w.Body.String())
	}
}
func TestOriginalPublishInterfaceNeedsNoRecipientFields(t *testing.T) {
	f := &officialClientStub{}
	g := officialRouter(f)
	before := time.Now().Unix()
	w := officialRequest(g, "publicMuxiOfficialMSG", `{"title":"survey","content":"please fill","extend_fields":{"url":"https://example.com/survey"},"later_time":60}`, "admin")
	if w.Code != 200 || f.published == nil {
		t.Fatalf("publish: %d %s", w.Code, w.Body.String())
	}
	msg := f.published.MuxiOfficialMSG
	if msg.Title != "survey" || msg.ExtendFields["url"] != "https://example.com/survey" || msg.PublicTime < before+60 {
		t.Fatalf("original message fields changed: %+v", msg)
	}
	f.publishErr = status.Error(codes.FailedPrecondition, "expired list")
	w = officialRequest(g, "publicMuxiOfficialMSG", `{"title":"survey"}`, "admin")
	if w.Code != 409 {
		t.Fatalf("missing list: %d %s", w.Code, w.Body.String())
	}
	var response map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response["code"] != float64(40801) {
		t.Fatalf("code=%v", response)
	}
}

func TestWhiteListHTTPExplainsUnavailableUserHistory(t *testing.T) {
	mr := miniredis.RunT(t)
	r := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer r.Close()
	f := &officialClientStub{}
	g := officialRouter(f, r)
	w := officialRequest(g, "setWhiteList", `{"strategy":"high_activity"}`, "admin")
	var response struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if w.Code != 409 || response.Code != 40804 || !strings.Contains(response.Msg, "尚未记录每日活跃学号") || f.selected != nil {
		t.Fatalf("not started: %d %s", w.Code, w.Body.String())
	}
	since := time.Now().Add(-time.Hour)
	if err := activity.Record(context.Background(), r, "a", since); err != nil {
		t.Fatal(err)
	}
	w = officialRequest(g, "setWhiteList", `{"strategy":"high_activity"}`, "admin")
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	earliest := activity.Day(since).AddDate(0, 0, 8).Format("2006-01-02")
	if w.Code != 409 || response.Code != 40804 || !strings.Contains(response.Msg, earliest) || f.selected != nil {
		t.Fatalf("warmup: %d %s", w.Code, w.Body.String())
	}
}
