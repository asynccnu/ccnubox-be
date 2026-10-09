package crawler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/asynccnu/ccnubox-be/common/pkg/metricsx"
	"github.com/prometheus/client_golang/prometheus"
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

func TestGetCurrentTeamResponseMetrics(t *testing.T) {
	for _, tc := range []struct {
		name     string
		body     string
		httpCode int
		result   string
	}{
		{"team", `{"status":true,"code":200,"data":{"id":"1","status":1}}`, 200, "success"},
		{"no team", `{"status":true,"code":200,"data":""}`, 200, "success"},
		{"missing id", `{"status":true,"code":200,"data":{"status":1}}`, 200, "invalid_response"},
		{"unknown status", `{"status":true,"code":200,"data":{"id":"1","status":3}}`, 200, "invalid_response"},
		{"invalid date", `{"status":true,"code":200,"data":{"id":"1","status":1,"onDate":"2026-02-30"}}`, 200, "invalid_response"},
		{"null data", `{"status":true,"code":200,"data":null}`, 200, "invalid_response"},
		{"auth error", `{"status":false,"code":20003,"data":""}`, 200, "auth_error"},
		{"http error", `unavailable`, 503, "http_error"},
		{"invalid json", `{`, 200, "network_or_decode_error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			registry := prometheus.NewRegistry()
			metrics := metricsx.NewWithRegisterer(registry, "team_test")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.httpCode)
				fmt.Fprint(w, tc.body)
			}))
			defer server.Close()
			c := NewReminderCrawler(server.Client(), time.Second, 20, 3, 100, metrics.Library)
			c.baseURL = server.URL
			_, err := c.GetCurrentTeam(context.Background(), "raw-jwt")
			if (err == nil) != (tc.result == "success") {
				t.Fatalf("result=%s err=%v", tc.result, err)
			}
			families, err := registry.Gather()
			if err != nil {
				t.Fatal(err)
			}
			var requests float64
			var durations uint64
			for _, family := range families {
				for _, metric := range family.GetMetric() {
					switch family.GetName() {
					case "team_test_library_upstream_requests_total":
						for _, label := range metric.GetLabel() {
							if label.GetName() == "endpoint" && label.GetValue() != "current_team" {
								t.Fatalf("unexpected endpoint: %s", label.GetValue())
							}
							if label.GetName() == "result" && label.GetValue() != tc.result {
								t.Fatalf("result=%s want=%s", label.GetValue(), tc.result)
							}
						}
						requests += metric.GetCounter().GetValue()
					case "team_test_library_upstream_duration_seconds":
						durations += metric.GetHistogram().GetSampleCount()
					}
				}
			}
			if requests != 1 || durations != 1 {
				t.Fatalf("requests=%v durations=%d", requests, durations)
			}
		})
	}
}

func TestDecodeInvitationTeamStrict(t *testing.T) {
	// 状态 7 仅用于结构校验，真正白名单由服务配置经契约核验后提供。
	valid := `{"id":2102744440918429696,"status":0,"username":"operator","isMasterUser":true,"expirationTime":"2026-09-23 23:58:36","userInfoList":[{"username":"A","status":7}]}`
	team, err := decodeInvitationTeam([]byte(valid))
	if err != nil || team.ID != "2102744440918429696" || team.Members["A"] != 7 || team.ExpirationTime.Hour() != 23 {
		t.Fatalf("team=%+v err=%v", team, err)
	}
	for _, raw := range []string{
		`null`, `{}`, `[]`,
		strings.Replace(valid, `"isMasterUser":true,`, "", 1),
		strings.Replace(valid, `"isMasterUser":true`, `"isMasterUser":null`, 1),
		strings.Replace(valid, `"username":"operator"`, `"username":123`, 1),
		strings.Replace(valid, `"expirationTime":"2026-09-23 23:58:36"`, `"expirationTime":""`, 1),
		strings.Replace(valid, `"status":7`, `"status":null`, 1),
		strings.Replace(valid, `"status":7`, `"status":"7"`, 1),
		strings.Replace(valid, `[{"username":"A","status":7}]`, `null`, 1),
		strings.Replace(valid, `[{"username":"A","status":7}]`, `[{"username":"A","status":7},{"username":"A","status":1}]`, 1),
	} {
		if _, err := decodeInvitationTeam([]byte(raw)); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
	if team, err := decodeInvitationTeam([]byte(`""`)); err != nil || team != nil {
		t.Fatalf("empty=%v err=%v", team, err)
	}
}

func TestInvitationUsesStrictBusinessSuccessAndRawToken(t *testing.T) {
	for _, body := range []string{`{"status":true,"code":200,"data":""}`, `{"status":true,"code":0,"data":""}`, `{"status":false,"code":200,"data":""}`, `{"status":true,"code":200,"data":null}`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "raw-jwt" || r.URL.Path != reminderTeamPath {
				t.Errorf("request=%v", r)
			}
			fmt.Fprint(w, body)
		}))
		c := NewReminderCrawler(server.Client(), time.Second, 20, 3, 100)
		c.baseURL = server.URL
		_, err := c.GetCurrentTeamForInvitation(context.Background(), "raw-jwt")
		if (err == nil) != (body == `{"status":true,"code":200,"data":""}`) {
			t.Errorf("body=%s err=%v", body, err)
		}
		server.Close()
	}
}
