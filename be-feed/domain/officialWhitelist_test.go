package domain

import (
	"testing"
	"time"
)

func TestCurrentStudentGraduationBoundary(t *testing.T) {
	zone := time.FixedZone("Asia/Shanghai", 8*60*60)
	for _, tc := range []struct {
		name string
		at   time.Time
		id   string
		want bool
	}{
		{"before_graduation", time.Date(2026, 6, 29, 23, 59, 59, 0, zone), "20220001", true},
		{"graduation_day", time.Date(2026, 6, 30, 0, 0, 0, 0, zone), "20220001", false},
		{"four_current_cohorts", time.Date(2026, 10, 20, 0, 0, 0, 0, zone), "20230001", true},
		{"new_cohort", time.Date(2026, 10, 20, 0, 0, 0, 0, zone), "20260001", true},
		{"future_cohort", time.Date(2026, 10, 20, 0, 0, 0, 0, zone), "20270001", false},
		{"before_new_cohort", time.Date(2026, 6, 29, 0, 0, 0, 0, zone), "20260001", false},
		{"new_year_no_early_graduation", time.Date(2027, 1, 1, 0, 0, 0, 0, zone), "20230001", true},
		{"utc_boundary", time.Date(2026, 6, 29, 16, 0, 0, 0, time.UTC), "20220001", false},
		{"bad_prefix", time.Date(2026, 10, 20, 0, 0, 0, 0, zone), "abcd0001", false},
		{"bad_identifier", time.Date(2026, 10, 20, 0, 0, 0, 0, zone), "2023 001", false},
		{"bare_year", time.Date(2026, 10, 20, 0, 0, 0, 0, zone), "2026", false},
		{"old_cohort", time.Date(2026, 10, 20, 0, 0, 0, 0, zone), "20190001", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsCurrentStudent(tc.id, tc.at); got != tc.want {
				t.Fatalf("got=%v want=%v", got, tc.want)
			}
		})
	}
}
