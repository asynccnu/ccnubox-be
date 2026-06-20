package crawler

import (
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
