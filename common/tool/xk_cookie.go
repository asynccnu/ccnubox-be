package tool

import ccnuv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/ccnu/v1"

// XKSystem 明确 Cookie 所属教务系统，不与图书馆 Token 共用。
type XKSystem string

const (
	XKUndergraduate XKSystem = "undergraduate"
	XKPostgraduate  XKSystem = "postgraduate"
)

// ResolveXKSystem 仅供兼容入口使用，内部调用必须传入明确系统。
func ResolveXKSystem(studentID, raw string) (XKSystem, error) {
	system := XKSystem(raw)
	switch raw {
	case "":
		return XKSystemForStudent(studentID)
	case "new":
		// 历史上只有培养方案使用 new，实际访问的是本科系统。
		system = XKUndergraduate
	}
	return system, ValidateXKSystem(studentID, system)
}

// XKSystemForStudent 供同时支持两类学生的业务选择对应系统。
func XKSystemForStudent(studentID string) (XKSystem, error) {
	switch ParseStudentType(studentID) {
	case UnderGraduate:
		return XKUndergraduate, nil
	case PostGraduate:
		return XKPostgraduate, nil
	default:
		return "", ccnuv1.ErrorInvalidXkSystem("无法识别学号对应的教务系统")
	}
}

func ValidateXKSystem(studentID string, system XKSystem) error {
	if err := CheckXKSystem(system); err != nil {
		return err
	}
	studentType := ParseStudentType(studentID)
	if (system == XKUndergraduate && studentType == UnderGraduate) ||
		(system == XKPostgraduate && studentType == PostGraduate) {
		return nil
	}
	return ccnuv1.ErrorInvalidXkSystem("教务系统与学号类型不匹配")
}

func CheckXKSystem(system XKSystem) error {
	switch system {
	case XKUndergraduate, XKPostgraduate:
		return nil
	default:
		return ccnuv1.ErrorInvalidXkSystem("不支持的教务系统类型")
	}
}
