package domain

import (
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

var schoolTimeZone = time.FixedZone("Asia/Shanghai", 8*60*60)

// IsCurrentStudent follows the four-year rule for official-message recipients.
// The academic cohort advances at 00:00 on June 30 in Asia/Shanghai: before
// that date the graduating fourth-year cohort remains enrolled.
func IsCurrentStudent(studentID string, at time.Time) bool {
	if len(studentID) < 5 || len(studentID) > 255 || !utf8.ValidString(studentID) || strings.TrimSpace(studentID) != studentID || strings.ContainsAny(studentID, "\x00\r\n\t ") {
		return false
	}
	for _, digit := range studentID[:4] {
		if digit < '0' || digit > '9' {
			return false
		}
	}
	enrolled, err := strconv.Atoi(studentID[:4])
	if err != nil || enrolled < 1900 {
		return false
	}
	at = at.In(schoolTimeZone)
	cohortYear := at.Year()
	graduation := time.Date(cohortYear, time.June, 30, 0, 0, 0, 0, schoolTimeZone)
	if at.Before(graduation) {
		cohortYear--
	}
	age := cohortYear - enrolled
	return age >= 0 && age < 4
}
