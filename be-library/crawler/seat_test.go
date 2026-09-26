package crawler

import (
	"testing"
	"time"

	"github.com/asynccnu/ccnubox-be/be-library/tool"
)

func TestDeriveDefaultSeatDate(t *testing.T) {
	loc := tool.GetLocation()

	daytime := time.Date(2026, 9, 10, 12, 0, 0, 0, loc)
	if got := deriveDefaultSeatDate(daytime); got != "2026-09-10" {
		t.Fatalf("daytime date = %s, want 2026-09-10", got)
	}

	beforeCutoff := time.Date(2026, 9, 10, 21, 59, 0, 0, loc)
	if got := deriveDefaultSeatDate(beforeCutoff); got != "2026-09-10" {
		t.Fatalf("before cutoff date = %s, want 2026-09-10", got)
	}

	night := time.Date(2026, 9, 10, 22, 30, 0, 0, loc)
	if got := deriveDefaultSeatDate(night); got != "2026-09-11" {
		t.Fatalf("night date = %s, want 2026-09-11", got)
	}
}
