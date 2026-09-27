package crawler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetCurrentTeamHTTP(t *testing.T) {
	body := `{"status":true,"code":200,"data":{"id":2102744440918429696,"status":1,"onDate":"2026-09-23"}}`
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodGet || r.URL.Path != reminderTeamPath || r.Header.Get("Authorization") != "raw-jwt" || r.Header.Get("token") != "" || r.Header.Get("X-hmac-request-key") != "" {
			t.Errorf("unexpected request: %s %s %v", r.Method, r.URL.Path, r.Header)
		}
		fmt.Fprint(w, body)
	}))
	defer server.Close()
	c := NewReminderCrawler(server.Client(), time.Second, 20, 3, 100)
	c.baseURL = server.URL
	team, err := c.GetCurrentTeam(context.Background(), "raw-jwt")
	if err != nil || team == nil || team.ID != "2102744440918429696" {
		t.Fatalf("team=%+v err=%v", team, err)
	}
	if calls != 1 {
		t.Fatalf("extra signing request: %d", calls)
	}
}

func TestDecodeReminderTeamRejectsUnknown(t *testing.T) {
	for _, raw := range []string{`null`, `[]`, `{}`, `"x"`, `{"id":"1"}`, `{"id":"1","status":null}`, `{"id":"1","status":3}`, `{"id":"1","status":1,"onDate":"2026-02-30"}`, `{"id":"1","status":1,"expirationTime":null}`} {
		if _, err := decodeReminderTeam([]byte(raw)); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
	team, err := decodeReminderTeam([]byte(`""`))
	if err != nil || team != nil {
		t.Fatalf("empty team=%+v err=%v", team, err)
	}
}

func TestTeamAuthClassificationDoesNotRefreshSeatHMAC(t *testing.T) {
	err := &upstreamError{Endpoint: "current_team", HTTPCode: 200, Code: 20003}
	if ClassifyUpstreamError(err) != "auth_error" || isAuthRejection(err) {
		t.Fatalf("20003 should classify as team auth failure, not seat HMAC rejection")
	}
}
