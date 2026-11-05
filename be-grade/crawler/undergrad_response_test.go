package crawler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestParseGradeResponse(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantCount int
		wantError string
	}{
		{
			name:      "valid response",
			body:      `{"code":0,"msg":"success","data":[{"cj0708id":"grade-id","zcj":80}]}`,
			wantCount: 1,
		},
		{
			name:      "valid empty response",
			body:      `{"code":0,"msg":"success","data":[]}`,
			wantCount: 0,
		},
		{
			name:      "business error",
			body:      `{"code":500,"msg":"failed","data":null}`,
			wantError: "code: 500",
		},
		{
			name:      "null data",
			body:      `{"code":0,"msg":"success","data":null}`,
			wantError: "null grade data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grades, err := parseGradeResponse([]byte(tt.body))
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("parseGradeResponse() error = %v, want containing %q", err, tt.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseGradeResponse() error = %v", err)
			}
			if len(grades) != tt.wantCount {
				t.Fatalf("parseGradeResponse() count = %d, want %d", len(grades), tt.wantCount)
			}
		})
	}
}

type detailTransport func(*http.Request) (*http.Response, error)

func (f detailTransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestGetDetailDistinguishesContentAndDependencyFailures(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{"invalid content", http.StatusOK, "<html>no score</html>", ErrDetailParse},
		{"unauthorized", http.StatusUnauthorized, "", ErrCookieTimeout},
		{"login", http.StatusOK, Login_URL, ErrCookieTimeout},
		{"upstream failure", http.StatusServiceUnavailable, "maintenance", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ug, err := NewUnderGrad(&http.Client{Transport: detailTransport(func(req *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body)), Request: req}, nil
			})})
			if err != nil {
				t.Fatal(err)
			}
			_, err = ug.GetDetail(context.Background(), "student", "class", "grade", 90)
			if err == nil {
				t.Fatal("expected detail failure")
			}
			if tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("err=%v want=%v", err, tc.want)
			}
			if tc.want == nil && (errors.Is(err, ErrDetailParse) || errors.Is(err, ErrCookieTimeout)) {
				t.Fatalf("dependency failure classified as permanent content failure: %v", err)
			}
		})
	}
}

func TestParseGradeResponseRejectsMissingOrAmbiguousScores(t *testing.T) {
	// 缺失、null 和文本状态不能当作有效零分；不推测等级制映射。
	for _, tc := range []struct {
		name string
		item string
		want float32
		ok   bool
	}{
		{"zero", `{"zcj":0}`, 0, true},
		{"fraction", `{"zcj":89.25,"zcjstr":"89.25"}`, 89.25, true},
		{"missing", `{}`, 0, false},
		{"null", `{"zcj":null}`, 0, false},
		{"text", `{"zcj":"优秀"}`, 0, false},
		{"publication text", `{"zcj":0,"zcjstr":"缓考"}`, 0, false},
		{"grade scale text", `{"zcj":90,"zcjstr":"优秀"}`, 90, false},
		{"different display", `{"zcj":89.25,"zcjstr":"89"}`, 89.25, true},
		{"float32 overflow", `{"zcj":1e40}`, 0, false},
		{"object", `{"zcj":{}}`, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			grades, err := parseGradeResponse([]byte(`{"code":0,"data":[` + tc.item + `]}`))
			if (err == nil) != tc.ok {
				t.Fatalf("grades=%+v err=%v want success=%v", grades, err, tc.ok)
			}
			if tc.ok && (len(grades) != 1 || grades[0].ZCJ != tc.want) {
				t.Fatalf("grades=%+v want score=%v", grades, tc.want)
			}
		})
	}
	if grades, err := parseGradeResponse([]byte(`{"code":0,"data":[{"zcj":80},{}]}`)); err == nil || len(grades) != 0 {
		t.Fatalf("partial invalid snapshot accepted: grades=%+v err=%v", grades, err)
	}
}

func TestParseGraduateResponseRejectsInvalidSnapshots(t *testing.T) {
	for _, tc := range []struct {
		body      string
		wantCount int
		ok        bool
	}{
		{`{"items":[{"cj":"0"}]}`, 1, true},
		{`{"items":[{"cj":"89.25"}]}`, 1, true},
		{`{"items":[]}`, 0, true},
		{`{"items":null}`, 0, false},
		{`{}`, 0, false},
		{`{"items":[{}]}`, 0, false},
		{`{"items":[{"cj":"优秀"}]}`, 0, false},
		{`{"items":[{"cj":"NaN"}]}`, 0, false},
		{`{"items":[{"cj":"80"},{"cj":"未发布"}]}`, 0, false},
		{`{"items":[{"cj":80}]}`, 0, false},
		{`<html>login</html>`, 0, false},
	} {
		grades, err := parseGraduateResponse([]byte(tc.body))
		if (err == nil) != tc.ok || (tc.ok && len(grades) != tc.wantCount) {
			t.Fatalf("body=%s grades=%+v err=%v want success=%v count=%d", tc.body, grades, err, tc.ok, tc.wantCount)
		}
	}
}
