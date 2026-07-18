package estimation

import "time"

// GetAcademicInfo 返回给定日期对应的学年和学期, 对齐 EstimateDateRange 的区间边界，间隙日期返回下一个学期
func GetAcademicInfo(t time.Time) (academicYear int, semester int) {
	y, m, d := t.Year(), t.Month(), t.Day()

	switch {
	// Semester 1: year-09-01 ~ year+1-01-15
	case m >= 9 || (m == 1 && d <= 15):
		if m >= 9 {
			academicYear = y
		} else {
			academicYear = y - 1
		}
		semester = 1

	// Semester 2 + 间隙 Jan 16 ~ Feb 14: year+1-01-16 ~ year+1-06-30
	case (m == 1 && d >= 16) || (m >= 2 && m <= 6):
		academicYear = y - 1
		semester = 2

	// Semester 3: year+1-07-01 ~ year+1-07-15
	case m == 7 && d <= 15:
		academicYear = y - 1
		semester = 3

	// 间隙 Jul 16 ~ Aug 31 → 下一学年的 semester 1
	default:
		academicYear = y
		semester = 1
	}
	return
}

// EstimateDateRange 根据学年和学期推算大致的起止日期
func EstimateDateRange(year, semester int) (startDate, endDate string) {
	loc := time.UTC
	switch semester {
	case 1:
		return time.Date(year, 9, 1, 0, 0, 0, 0, loc).Format("2006-01-02"),
			time.Date(year+1, 1, 15, 0, 0, 0, 0, loc).Format("2006-01-02")
	case 2:
		return time.Date(year+1, 2, 15, 0, 0, 0, 0, loc).Format("2006-01-02"),
			time.Date(year+1, 6, 30, 0, 0, 0, 0, loc).Format("2006-01-02")
	case 3:
		return time.Date(year+1, 7, 1, 0, 0, 0, 0, loc).Format("2006-01-02"),
			time.Date(year+1, 7, 15, 0, 0, 0, 0, loc).Format("2006-01-02")
	default:
		return "", ""
	}
}
