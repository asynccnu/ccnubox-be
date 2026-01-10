package errorx

import (
	"fmt"
	"io"
	"runtime"
	"strings"
)

type chainError struct {
	msg   string
	file  string
	line  int
	funcN string
	cause error
}

// Error 保持简洁，用于日志单行打印
func (e *chainError) Error() string {
	if e.cause == nil {
		return e.msg
	}
	return fmt.Sprintf("%s: %v", e.msg, e.cause)
}

func (e *chainError) Unwrap() error {
	return e.cause
}

// Format 实现 fmt.Formatter 接口，核心逻辑在这里
func (e *chainError) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			// %+v 模式：多行堆栈
			fmt.Fprintf(s, "%s", e.msg)
			fmt.Fprintf(s, "\n\t%s:%d %s", e.file, e.line, e.funcN)
			if e.cause != nil {
				fmt.Fprintf(s, "\n%+v", e.cause) // 递归打印下一层
			}
			return
		}
		// 普通 %v
		io.WriteString(s, e.Error())
	case 's':
		io.WriteString(s, e.Error())
	}
}

// New 创建错误
func New(message string) error {
	file, line, fn := getCallerInfo(2)
	return &chainError{
		msg:   message,
		file:  file,
		line:  line,
		funcN: fn,
	}
}

// Wrap 包装错误
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	file, line, fn := getCallerInfo(2)
	return &chainError{
		msg:   message,
		file:  file,
		line:  line,
		funcN: fn,
		cause: err,
	}
}

// Errorf 支持格式化并自动识别 %w
func Errorf(format string, args ...any) error {
	var cause error
	for _, arg := range args {
		if err, ok := arg.(error); ok {
			cause = err
			break
		}
	}

	msg := fmt.Sprintf(strings.ReplaceAll(format, "%w", "%v"), args...)
	// 如果是包装错误，去掉重复的子错误文本
	if cause != nil {
		msg = strings.Split(msg, cause.Error())[0]
		msg = strings.TrimSuffix(msg, ": ")
	}

	file, line, fn := getCallerInfo(2)
	return &chainError{
		msg:   msg,
		file:  file,
		line:  line,
		funcN: fn,
		cause: cause,
	}
}

func getCallerInfo(skip int) (string, int, string) {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown", 0, "unknown"
	}
	// 只保留文件名，不保留全路径（可选）
	if lastSlash := strings.LastIndexAny(file, "/\\"); lastSlash != -1 {
		file = file[lastSlash+1:]
	}
	fn := runtime.FuncForPC(pc).Name()
	if lastDot := strings.LastIndex(fn, "."); lastDot != -1 {
		fn = fn[lastDot+1:]
	}
	return file, line, fn
}

func FormatGRPCErrorFunc(grpc error) func(origin error) error {
	return func(origin error) error {
		return Wrap(grpc, fmt.Sprintf("%+v", origin))
	}
}
