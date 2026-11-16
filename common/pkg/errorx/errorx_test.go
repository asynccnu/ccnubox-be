package errorx

import (
	"fmt"
	"strings"
	"testing"

	"github.com/go-kratos/kratos/v2/errors"
)

func TestError(t *testing.T) {
	err := New("test")
	err1 := fmt.Errorf("测试: %w", err)
	if !errors.Is(err1, err) {
		t.Log("不相等")
	}

	err2 := Errorf("测试: %w", err1)
	if !errors.Is(err2, err) {
		t.Log("不相等")
	}

	err3 := Errorf("再测试一下: %w 测试两下: %d", err2, 1)
	fmt.Println()
	fmt.Println(err2.Error())
	fmt.Println()
	fmt.Printf("%+v\n", err3)

	// fmt.Sprintf("%+v", err3)
}

func TestErrorfFormat(t *testing.T) {
	cause := New("boom")

	cases := []struct {
		name      string
		format    string
		args      []any
		want      string
		wantCount int
	}{
		{
			name:      "wrap followed by more text",
			format:    "a: %w; b: %v",
			args:      []any{cause, 42},
			want:      "a: boom; b: 42",
			wantCount: 1,
		},
		{
			name:      "wrap in the middle keeps trailing literal",
			format:    "step 1 %w then step 2 done",
			args:      []any{cause},
			want:      "step 1 boom then step 2 done",
			wantCount: 1,
		},
		{
			name:      "cause without wrap keeps all verbs",
			format:    "failed: %v, code: %d",
			args:      []any{cause, 7},
			want:      "failed: boom, code: 7",
			wantCount: 1,
		},
		{
			name:      "plain text without args",
			format:    "plain message",
			want:      "plain message",
			wantCount: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Errorf(tc.format, tc.args...)
			msg := err.(*customError).msg
			if msg != tc.want {
				t.Fatalf("msg = %q, want %q", msg, tc.want)
			}
			if strings.Contains(msg, "%!") {
				t.Fatalf("msg contains invalid format marker: %q", msg)
			}
			// Error() 里 cause 文本只能出现一次：Errorf 已经渲染进 msg，不再重复拼接。
			got := err.Error()
			if got != tc.want {
				t.Fatalf("Error() = %q, want %q", got, tc.want)
			}
			if count := strings.Count(got, "boom"); count != tc.wantCount {
				t.Fatalf("Error() contains cause %d times, want %d: %q", count, tc.wantCount, got)
			}
		})
	}

	t.Run("cause keeps errors.Is chain", func(t *testing.T) {
		err := Errorf("a: %w; b: %v", cause, 42)
		if !errors.Is(err, cause) {
			t.Fatal("expected errors.Is to find cause")
		}
	})

	t.Run("cause without verb is still appended once", func(t *testing.T) {
		err := Errorf("no verb here", cause)
		if got, want := err.Error(), "no verb here: boom"; got != want {
			t.Fatalf("Error() = %q, want %q", got, want)
		}
		if got := strings.Count(err.Error(), "boom"); got != 1 {
			t.Fatalf("Error() contains cause %d times, want 1", got)
		}
	})
}

func TestErrorfVerbScanning(t *testing.T) {
	cause := New("boom")

	cases := []struct {
		name    string
		format  string
		args    []any
		wantMsg string
		wantErr string
	}{
		{
			// %%w 是字面量，不能被改写成 %%v，也不能因此认为 cause 已渲染。
			name:    "escaped percent then wrap",
			format:  "literal %%w: %w",
			args:    []any{cause},
			wantMsg: "literal %w: boom",
			wantErr: "literal %w: boom",
		},
		{
			name:    "explicit index before wrap",
			format:  "failed: %[1]w",
			args:    []any{cause},
			wantMsg: "failed: boom",
			wantErr: "failed: boom",
		},
		{
			// cause 是第 1 个参数但只被第 2 个占位符消费，Error() 必须补上 cause 文本。
			name:    "cause argument not consumed",
			format:  "code: %[2]d",
			args:    []any{cause, 7},
			wantMsg: "code: 7",
			wantErr: "code: 7: boom",
		},
		{
			name:    "escaped percent does not break later wrap",
			format:  "100%% done: %w",
			args:    []any{cause},
			wantMsg: "100% done: boom",
			wantErr: "100% done: boom",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Errorf(tc.format, tc.args...)
			if got := err.(*customError).msg; got != tc.wantMsg {
				t.Fatalf("msg = %q, want %q", got, tc.wantMsg)
			}
			if got := err.Error(); got != tc.wantErr {
				t.Fatalf("Error() = %q, want %q", got, tc.wantErr)
			}
			if got := strings.Count(err.Error(), "boom"); got != 1 {
				t.Fatalf("Error() contains cause %d times, want 1: %q", got, err.Error())
			}
		})
	}
}

func TestWrapAppendsCauseOnce(t *testing.T) {
	cause := New("boom")

	wrapped := Wrap(cause, "layer")
	if got, want := wrapped.Error(), "layer: boom"; got != want {
		t.Fatalf("Wrap().Error() = %q, want %q", got, want)
	}
	if got := strings.Count(wrapped.Error(), "boom"); got != 1 {
		t.Fatalf("Wrap().Error() contains cause %d times, want 1", got)
	}
	if !errors.Is(wrapped, cause) {
		t.Fatal("expected errors.Is to find cause")
	}
}

func TestErrorfStackFormat(t *testing.T) {
	err := Errorf("x: %w", New("boom"))

	out := fmt.Sprintf("%+v", err)
	if count := strings.Count(out, "boom"); count != 1 {
		t.Fatalf("%%+v contains cause %d times, want 1: %s", count, out)
	}
	// 本层与 cause 层都是 *customError，位置必须逐层保留。
	if got := strings.Count(out, "errorx_test.go"); got < 2 {
		t.Fatalf("%%+v keeps %d positions, want at least 2 (current layer + cause): %s", got, out)
	}

	// FormatErrorFunc 的用户可见错误由 %+v 组装，也不能重复 cause 文本。
	visible := FormatErrorFunc(New("visible layer"))(err)
	if count := strings.Count(visible.Error(), "boom"); count != 1 {
		t.Fatalf("FormatErrorFunc error contains cause %d times, want 1: %s", count, visible.Error())
	}
}

func TestWrapStackFormat(t *testing.T) {
	wrapped := Wrap(New("boom"), "layer")

	out := fmt.Sprintf("%+v", wrapped)
	if !strings.Contains(out, "layer") {
		t.Fatalf("%%+v lost wrap layer text: %s", out)
	}
	if count := strings.Count(out, "boom"); count != 1 {
		t.Fatalf("%%+v contains cause %d times, want 1: %s", count, out)
	}
	if got := strings.Count(out, "errorx_test.go"); got < 2 {
		t.Fatalf("%%+v keeps %d positions, want at least 2 (current layer + cause): %s", got, out)
	}
}
