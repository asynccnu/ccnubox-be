package errorx

import "runtime"

// GetCallerInfo 获取调用信息
func GetCallerInfo(skip int) (string, int, string) {
	// skip: 调用栈层级，1 表示当前函数，2 表示上层调用函数,3表示上层函数(一般用3,因为要额外包一层)
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown", 0, "unknown"
	}
	function := runtime.FuncForPC(pc).Name()
	return file, line, function
}
