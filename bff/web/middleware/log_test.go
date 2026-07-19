package middleware

import (
	"errors"
	"fmt"
	"testing"

	b_errorx "github.com/asynccnu/ccnubox-be/bff/pkg/errorx"
)

func TestFindCustomErrorTraversesEntireChain(t *testing.T) {
	want := b_errorx.New(500, 50101, "banner failed")
	wrapped := fmt.Errorf("outer: %w", fmt.Errorf("middle: %w", want))

	got, ok := findCustomError(wrapped)
	if !ok {
		t.Fatal("CustomError was not found in a nested error chain")
	}
	var expected *b_errorx.CustomError
	if !errors.As(want, &expected) {
		t.Fatal("test setup did not create a CustomError")
	}
	if got != expected {
		t.Fatalf("got CustomError %p, want %p", got, expected)
	}
}

func TestFindCustomErrorRejectsUnknownError(t *testing.T) {
	if got, ok := findCustomError(errors.New("unknown")); ok || got != nil {
		t.Fatalf("findCustomError() = (%v, %v), want (nil, false)", got, ok)
	}
}
