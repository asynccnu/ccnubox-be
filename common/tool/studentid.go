package tool

import (
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type StudentType int

const (
	Unknown       StudentType = iota
	PostGraduate              // 研究生 (1 或 0)
	UnderGraduate             // 本科生 (2)
)

// IsValidStudentID 校验内部学号标识的存储边界，不限制具体学制和学号格式。
// 提醒订阅表最多存储 64 字节，生产与消费偏好时必须使用相同约束。
func IsValidStudentID(studentID string) bool {
	return studentID != "" && len(studentID) <= 64 && strings.TrimSpace(studentID) == studentID && utf8.ValidString(studentID)
}

// ParseStudentType 根据学号规则解析学生类型 区分是学号第五位，本科是2，硕士是1，博士是0，工号是6或9
func ParseStudentType(studentId string) StudentType {
	if len(studentId) <= 4 {
		return Unknown
	}
	// 学号第五位即 studentId[4]
	switch studentId[4] {
	case '0', '1': // 实际上0代表博士生?
		return PostGraduate
	case '2':
		return UnderGraduate
	default:
		return Unknown
	}
}

// IsGraduated 根据学号判断是否毕业
// 规则：学号前4位为入学年份
// 当前年份 - 入学年份 >= 5 认为已毕业
func IsGraduated(studentId string) bool {
	// 学号长度不足4位
	if len(studentId) < 4 {
		return false
	}

	// 取前4位作为年份
	yearStr := studentId[:4]

	// 转换为整数
	enrollYear, err := strconv.Atoi(yearStr)
	if err != nil {
		return false
	}

	// 获取当前年份
	currentYear := time.Now().Year()

	// 判断是否毕业
	if currentYear-enrollYear >= 5 {
		return true
	}

	return false
}
