package tool

import (
	"strconv"
	"time"
)

// GetCurrentAcademicYearAndSemester 根据时间粗略获取当前学年和学期
func GetCurrentAcademicYearAndSemester(now time.Time) (int, int) {
	year := now.Year()
	month := int(now.Month())

	switch {
	case month >= 9: // 9-12月
		return year, 1

	case month == 8: // 8月
		return year, 1

	case month == 1: // 1月
		return year - 1, 1

	case month >= 2 && month <= 7: // 2-7月
		return year - 1, 2
	}

	return year, 1 // 理论不会走到这里
}

func GetCurrentAcademicYearAndSemesterStr(now time.Time) (string, string) {
	y, s := GetCurrentAcademicYearAndSemester(now)
	return strconv.Itoa(y), strconv.Itoa(s)
}
