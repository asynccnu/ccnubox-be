package semesterInfo

import "time"

// SemesterInfo 学期信息
type SemesterInfo struct {
	Year       int // 学年开始年份
	Semester   int // 学期编号
	WeekNumber int // 当前是第几周
}

// GetSemesterInfo 计算学年、学期、当前周
// startDate: 开学第一天 (yyyy-MM-dd)
func GetSemesterInfo(startDateStr string) (*SemesterInfo, error) {
	const layout = "2006-01-02"
	startDate, err := time.Parse(layout, startDateStr)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	year := startDate.Year()
	month := startDate.Month()
	var academicYear int
	var semester int

	// 判断学年和学期
	switch {
	case month >= 9 && month <= 12: // 9-12月
		academicYear = year
		semester = 1
	case month >= 1 && month <= 2: // 1-2月
		academicYear = year - 1
		semester = 1
	case month >= 3 && month <= 6: // 3-6月
		academicYear = year - 1
		semester = 2
	case month >= 7 && month <= 8: // 7-8月
		academicYear = year - 1
		semester = 3
	}

	// 计算当前周数
	// 开学第一天一般是星期一，第一周为1
	daysSinceStart := int(now.Sub(startDate).Hours() / 24)
	weekNumber := (daysSinceStart / 7) + 1
	if weekNumber < 1 {
		weekNumber = 1
	}

	return &SemesterInfo{
		Year:       academicYear,
		Semester:   semester,
		WeekNumber: weekNumber,
	}, nil
}
