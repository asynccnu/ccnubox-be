package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/asynccnu/ccnubox-be/bff/cron"
	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/service/activity"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

type activityJWTHandler struct{ ijwt.Handler }

func (activityJWTHandler) JWTKey() []byte                                  { return []byte("activity-test-key") }
func (activityJWTHandler) CheckSession(*gin.Context, string) (bool, error) { return false, nil }

func TestAuthenticationRequiresDurableActivityAdmission(t *testing.T) {
	mr := miniredis.RunT(t)
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, activity.China)
	mr.SetTime(now)
	r := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: -1})
	defer r.Close()
	h := activityJWTHandler{}
	m := NewLoginMiddleWare(h, r)
	m.now = func() time.Time { return now }
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, ijwt.UserClaims{StudentId: "a"}).SignedString(h.JWTKey())
	if err != nil {
		t.Fatal(err)
	}
	called := 0
	g := gin.New()
	g.GET("/protected", m.MiddlewareFunc(), func(c *gin.Context) {
		called++
		claims, err := ginx.GetClaims[ijwt.UserClaims](c)
		if err != nil || claims.StudentId != "a" {
			t.Fatalf("claims=%+v err=%v", claims, err)
		}
		if !r.SIsMember(context.Background(), activity.Key(now)+":pending", "a").Val() {
			t.Fatal("handler ran before activity intent was persisted")
		}
		c.Status(http.StatusOK)
	})
	request := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		g.ServeHTTP(w, req)
		return w
	}
	mr.SetError("ERR activity admission unavailable")
	if w := request(); w.Code != http.StatusServiceUnavailable || called != 0 {
		t.Fatalf("untracked request admitted: status=%d called=%d", w.Code, called)
	}
	mr.SetError("")
	if w := request(); w.Code != http.StatusOK || called != 1 {
		t.Fatalf("recovered admission: status=%d called=%d", w.Code, called)
	}
	// An HLL failure cannot remove the durable exact intent.
	r.Set(context.Background(), cron.ActiveUsersBucketKey(now), "wrong-type", 0)
	metrics := NewPrometheusMiddleware(nil, r)
	if err := metrics.recordActiveUser(context.Background(), "a", now); err == nil {
		t.Fatal("expected approximate metric error")
	}
	if err := activity.EnsureDay(context.Background(), r, now.AddDate(0, 0, 1)); err != nil {
		t.Fatal(err)
	}
	if !r.SIsMember(context.Background(), activity.Key(now), "a").Val() {
		t.Fatal("durable activity was not replayed")
	}
}
