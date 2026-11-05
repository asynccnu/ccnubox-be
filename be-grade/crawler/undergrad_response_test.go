package crawler

import (
	"context"
	"errors"
	"fmt"
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

func TestParseGradeResponseSeparatesUnavailableScores(t *testing.T) {
	// 数值成绩和等级展示可共存；缺失、文本状态不能当作有效零分。
	for _, tc := range []struct {
		name        string
		item        string
		want        float32
		unavailable bool
		wantErr     bool
	}{
		{"zero", `{"zcj":0}`, 0, false, false},
		{"fraction", `{"zcj":89.25,"zcjstr":"89.25"}`, 89.25, false, false},
		{"numeric string", `{"zcj":"89.25"}`, 89.25, false, false},
		{"missing", `{}`, 0, true, false},
		{"null", `{"zcj":null}`, 0, true, false},
		{"text", `{"zcj":"优秀"}`, 0, true, false},
		{"publication text", `{"zcj":0,"zcjstr":"缓考"}`, 0, true, false},
		{"unknown status", `{"zcj":0,"zcjstr":"未知状态"}`, 0, true, false},
		{"grade scale text", `{"zcj":90,"zcjstr":"优秀"}`, 90, false, false},
		{"different display", `{"zcj":89.25,"zcjstr":"89"}`, 89.25, false, false},
		{"float32 overflow", `{"zcj":1e40}`, 0, false, true},
		{"object", `{"zcj":{}}`, 0, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			grades, err := parseGradeResponse([]byte(`{"code":0,"data":[` + tc.item + `]}`))
			if (err != nil) != tc.wantErr {
				t.Fatalf("grades=%+v err=%v want error=%v", grades, err, tc.wantErr)
			}
			if err == nil && (len(grades) != 1 || grades[0].ZCJ != tc.want || grades[0].ScoreUnavailable != tc.unavailable) {
				t.Fatalf("grades=%+v want score=%v unavailable=%v", grades, tc.want, tc.unavailable)
			}
		})
	}
	grades, err := parseGradeResponse([]byte(`{"code":0,"data":[{"zcj":80},{}]}`))
	if err != nil || len(grades) != 2 || grades[0].ScoreUnavailable || !grades[1].ScoreUnavailable {
		t.Fatalf("one unavailable score blocked the whole page: grades=%+v err=%v", grades, err)
	}
}

func TestParseGraduateResponsePreservesScoreStates(t *testing.T) {
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
		{`{"items":[{}]}`, 1, true},
		{`{"items":[{"cj":"优秀"}]}`, 1, true},
		{`{"items":[{"cj":"NaN"}]}`, 1, true},
		{`{"items":[{"cj":"80"},{"cj":"未发布"}]}`, 2, true},
		{`{"items":[{"cj":80}]}`, 0, false},
		{`<html>login</html>`, 0, false},
	} {
		grades, err := parseGraduateResponse([]byte(tc.body))
		if (err == nil) != tc.ok || (tc.ok && len(grades) != tc.wantCount) {
			t.Fatalf("body=%s grades=%+v err=%v want success=%v count=%d", tc.body, grades, err, tc.ok, tc.wantCount)
		}
	}
}

func TestUndergradPaginationRequiresCompleteSnapshot(t *testing.T) {
	// 这些是协议单元测试，不代替上线前的真实上游分页样例核验。
	for _, name := range []string{"success", "second page failed", "repeated page", "changed total", "missing total", "missing total with grades", "empty", "missing total on second page"} {
		t.Run(name, func(t *testing.T) {
			calls := 0
			ug, _ := NewUnderGrad(&http.Client{Transport: detailTransport(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.Query().Get("kksj") != "" || req.URL.Query().Get("pageNum") != fmt.Sprint(calls) {
					t.Fatal("cross-term pagination parameters changed")
				}
				body := `{"code":0,"total":2,"data":[{"jx0404id":"old","xqmc":"2020-2021-1","zcj":0}]}`
				code := http.StatusOK
				if calls == 2 {
					body = `{"code":0,"total":2,"data":[{"jx0404id":"new","xqmc":"2025-2026-1","zcj":90}]}`
				}
				switch name {
				case "second page failed":
					if calls == 2 {
						code = http.StatusServiceUnavailable
					}
				case "repeated page":
					body = `{"code":0,"total":2,"data":[{"jx0404id":"old","zcj":0}]}`
				case "changed total":
					if calls == 2 {
						body = `{"code":0,"total":3,"data":[]}`
					}
				case "missing total":
					body = `{"code":0,"data":[]}`
				case "missing total with grades":
					body = `{"code":0,"data":[{"jx0404id":"old","zcj":80}]}`
				case "missing total on second page":
					if calls == 2 {
						body = `{"code":0,"data":[]}`
					}
				case "empty":
					body = `{"code":0,"total":0,"data":[]}`
				}
				return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
			})})
			snapshot, err := ug.GetGradeSnapshot(context.Background(), 0, 0, 1)
			grades := snapshot.Grades
			if name == "success" {
				if err != nil || !snapshot.Complete || len(grades) != 2 || calls != 2 || grades[0].ZCJ != 0 || grades[0].XQMC != "2020-2021-1" {
					t.Fatalf("grades=%v err=%v calls=%d", grades, err, calls)
				}
			} else if name == "empty" {
				if err != nil || !snapshot.Complete || len(grades) != 0 || calls != 1 {
					t.Fatal(grades, err, calls)
				}
			} else if name == "missing total" || name == "missing total with grades" {
				want := 0
				if name == "missing total with grades" {
					want = 1
				}
				if err != nil || snapshot.Complete || len(grades) != want || calls != 1 {
					t.Fatal(snapshot, err, calls)
				}
			} else if err == nil || len(grades) != 0 {
				t.Fatalf("partial snapshot accepted: %v %v", grades, err)
			}
		})
	}
}

func TestGraduatePaginationUsesAllTerms(t *testing.T) {
	calls := 0
	g, _ := NewGraduate(&http.Client{Transport: detailTransport(func(req *http.Request) (*http.Response, error) {
		calls++
		_ = req.ParseForm()
		if req.Form.Get("xnm") != "" || req.Form.Get("xqm") != "" || req.Form.Get("queryModel.currentPage") != fmt.Sprint(calls) {
			t.Fatal(req.Form)
		}
		body := fmt.Sprintf(`{"totalCount":2,"items":[{"jxb_id":"class-%d","cj":"0"}]}`, calls)
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
	})})
	grades, err := g.GetGraduateGrades(context.Background(), "cookie", 0, 0, 300)
	if err != nil || len(grades) != 2 || calls != 2 {
		t.Fatal(grades, err, calls)
	}
}

func TestGraduateResponseWithoutTotalIsReadableButIncomplete(t *testing.T) {
	g, _ := NewGraduate(&http.Client{Transport: detailTransport(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"items":[{"jxb_id":"a","cj":"80"}]}`)), Request: req}, nil
	})})
	snapshot, err := g.GetGraduateSnapshot(context.Background(), "cookie", 0, 0, 300)
	if err != nil || snapshot.Complete || len(snapshot.Grades) != 1 || snapshot.Grades[0].Cj != "80" {
		t.Fatal(snapshot, err)
	}
}
