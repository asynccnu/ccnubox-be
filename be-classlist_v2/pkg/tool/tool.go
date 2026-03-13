package tool

import (
	"strconv"
	"time"
)

func CheckSY(semester, year string) bool {
	var tag1, tag2 bool
	y, err := strconv.Atoi(year)
	currentYear := time.Now().Year()
	if err != nil || y < 2006 || y >= currentYear+2 { // 年份小于2006或者年份大于后年的不予处理
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
