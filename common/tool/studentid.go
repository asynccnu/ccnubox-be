package tool

type StudentType int

const (
	Unknown       StudentType = iota
	PostGraduate              // 研究生 (1 或 0)
	UnderGraduate             // 本科生 (2)
)

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
