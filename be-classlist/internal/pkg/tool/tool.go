package tool

import (
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func CheckSY(semester, year string) bool {

	var tag1, tag2 bool
	y, err := strconv.Atoi(year)
	currentYear := time.Now().Year()
	if err != nil || y < 2006 || y >= currentYear+2 { //年份小于2006或者年份大于后年的不予处理
		tag1 = false
	} else {
		tag1 = true
	}
	if semester == "1" || semester == "2" || semester == "3" {
		tag2 = true
	} else {
		tag2 = false
	}
	return tag1 && tag2

}
func ParseWeeks(weeks int64) []int {
	if weeks <= 0 {
		return []int{}
	}
	var weeksList []int
	for i := 1; (1 << (i - 1)) <= weeks; i++ {
		if weeks&(1<<(i-1)) != 0 {
			weeksList = append(weeksList, i)
		}
	}
	return weeksList
}
func FormatWeeks(weeks []int) string {
	if len(weeks) == 0 {
		return ""
	}

	// 对周数集合排序
	sort.Ints(weeks)

	var result strings.Builder
	start := weeks[0]
	end := start
	isSingle := start%2 != 0
	isMixed := false

	// 检查是否是单周、双周还是混合
	for _, week := range weeks {
		if (week%2 == 0) != !isSingle {
			isMixed = true
		}
	}

	// 遍历周数集合，生成格式化字符串
	for i := 1; i < len(weeks); i++ {
		if weeks[i] == end+1 {
			end = weeks[i]
		} else {
			if start == end {
				result.WriteString(strconv.Itoa(start))
			} else {
				result.WriteString(strconv.Itoa(start) + "-" + strconv.Itoa(end))
			}
			result.WriteString(",")
			start = weeks[i]
			end = start
		}
	}

	// 处理最后一段区间
	if start == end {
		result.WriteString(strconv.Itoa(start))
	} else {
		result.WriteString(strconv.Itoa(start) + "-" + strconv.Itoa(end))
	}

	// 添加 "(单)" 或 "(双)" 标识
	if !isMixed {
		if isSingle {
			result.WriteString("周(单)")
		} else {
			result.WriteString("周(双)")
		}
	} else {
		result.WriteString("周")
	}

	return result.String()
}
func CheckIfThisYear(xnm, xqm string) bool {
	y, _ := strconv.Atoi(xnm)
	s, _ := strconv.Atoi(xqm)
	currentYear := time.Now().Year()
	currentMonth := time.Now().Month()
	//currentYear := 2023
	//currentMonth := 10
	if currentMonth >= 9 {
		return (y == currentYear) && (s == 1)
	}
	if currentMonth <= 1 {
		return (y == currentYear-1) && (s == 1)
	}
	if currentMonth >= 2 && currentMonth <= 6 {
		return (y == currentYear-1) && (s == 2)
	}
	if currentMonth >= 7 && currentMonth <= 8 {
		return (y == currentYear-1) && (s == 3)
	}
	return false

}

// CheckIsUndergraduate 检查该学号是否是本科生
func CheckIsUndergraduate(stuId string) bool {
	return stuId[4] == '2'
	//区分是学号第五位，本科是2，硕士是1，博士是0，工号是6或9
}
func RandomBool(p float64) bool {
	// 生成 0 到 1 之间的随机浮点数
	const n int = 100000
	randomValue := rand.Intn(n) // 生成 [0.0, 1.0) 之间的随机数
	return randomValue < int(p*(float64(n)))
}

//func IsNeedCraw() bool {
//	currentTime := time.Now()
//	//考虑每个学期的情况，9月和2月的月末以及10月和3月的月初，课表容易变化
//	if currentTime.Month() == time.September || currentTime.Month() == time.February {
//		if currentTime.Weekday() == time.Monday && isLastOrFirstWeek(currentTime, false) {
//			return true
//		}
//	}
//	if currentTime.Month() == time.October || currentTime.Month() == time.March {
//		if currentTime.Weekday() == time.Monday && isLastOrFirstWeek(currentTime, true) {
//			return true
//		}
//	}
//	//其他时间默认有0.3的概率
//	return RandomBool(0.3)
//}
//func isLastOrFirstWeek(t time.Time, first bool) bool {
//	// 检查当前日期是否是周一
//	if t.Weekday() != time.Monday {
//		return false
//	}
//
//	// 获取当前月的信息
//	currentMonth := t.Month()
//
//	var week time.Time
//	if !first {
//		// 下一个周：在当前日期基础上加7天
//		week = t.AddDate(0, 0, 7)
//	} else {
//		// 上一周：在当前日期基础上减7天
//		week = t.AddDate(0, 0, -7)
//	}
//
//	// 获取下一个周或者上一周的月信息
//	WeekMonth := week.Month()
//
//	// 如果下一个或者上一周的月份和当前月份不同，那么今天是当前月的最后一个周或者第一周
//	return currentMonth != WeekMonth
//}

// IsExist 判断文件或者目录是否存在
func IsExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

func OpenFile(path string, name string) (*os.File, error) {
	var logfile *os.File
	var err error
	filename := filepath.Join(path, name)
	// 判断日志路径是否存在，如果不存在就创建
	if exist := IsExist(path); !exist {
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			return nil, err
		}
	}
	if exist := IsExist(filename); !exist {
		logfile, err = os.Create(filepath.Join(filename))
	} else {
		logfile, err = os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	}
	if err != nil {
		return nil, err
	}
	return logfile, nil
}

// FormatTimeInUTC 将 time.Time 转换为 UTC 时区的格式化时间字符串
func FormatTimeInUTC(t time.Time) string {
	// 获取 UTC 时区
	location := time.UTC

	// 将时间转换为 UTC 时区
	utcTime := t.In(location)

	// 格式化并返回，精确到微秒
	return utcTime.Format("2006-01-02T15:04:05.000000")
}
