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
			body:      `{"code":0,"msg":"success","data":[{"cj0708id":"grade-id"}]}`,
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
