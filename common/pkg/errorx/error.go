package errorx

import (
	"errors"
	"fmt"
	"io"
	"runtime"
	"strings"
)

type customError struct {
	msg   string
	file  string
	line  int
	funcN string
	cause error
	// causeRendered 表示 msg 里已经渲染过 cause 文本，Error() 不再重复拼接。
	causeRendered bool
}

// Error 保持简洁：只负责返回单行错误链
// 效果：层级1: 层级2: 底层错误
func (e *customError) Error() string {
	if e.cause == nil || e.causeRendered {
		return e.msg
	}
	return fmt.Sprintf("%s: %v", e.msg, e.cause)
}

func (e *customError) Unwrap() error {
	return e.cause
}

// Format 实现 fmt.Formatter 接口
// 只有在 fmt.Sprintf("%+v", err) 时才会触发递归堆栈打印
func (e *customError) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			// 1. 打印当前层消息
			fmt.Fprintf(s, "%s", e.msg)
			// 2. 打印当前层文件行号和函数名 (使用相对路径)
			fmt.Fprintf(s, "\n\t%s:%d %s", e.file, e.line, e.funcN)
			// 3. 递归打印子错误堆栈
			if e.cause != nil {
				if e.causeRendered {
					// msg 里已经渲染过 cause 文本，只补各层位置，避免文本重复。
					writeCausePositions(s, e.cause)
				} else {
					fmt.Fprintf(s, "\n%+v", e.cause)
				}
			}
			return
		}
		// 普通 %v
		io.WriteString(s, e.Error())
	case 's':
		io.WriteString(s, e.Error())
	}
}

// writeCausePositions 只打印 cause 链上各层的位置，不打印文本（文本已在上层 msg 中）。
func writeCausePositions(s fmt.State, err error) {
	for err != nil {
		cause, ok := err.(*customError)
		if !ok {
			return
		}
		fmt.Fprintf(s, "\n\t%s:%d %s", cause.file, cause.line, cause.funcN)
		err = cause.cause
	}
}

// New 创建一个新的错误记录点
func New(message string) error {
	file, line, fn := getCallerInfo(2)
	return &customError{
		msg:   message,
		file:  file,
		line:  line,
		funcN: fn,
	}
}

// Wrap 包装一个现有错误，增加当前层的上下文
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	file, line, fn := getCallerInfo(2)
	return &customError{
		msg:   message,
		file:  file,
		line:  line,
		funcN: fn,
		cause: err,
	}
}

func Unwrap(err error) error {
	return errors.Unwrap(err)
}

func Is(err, target error) bool {
	return errors.Is(err, target)
}

func As(err error, target any) bool {
	return errors.As(err, target)
}

// Errorf 格式化并识别 %w
// 优化：msg 字段只保留当前层的格式化结果，不合并子错误文本
func Errorf(format string, args ...any) error {
	var cause error

	// 1. 找到第一个 error 类型的参数作为 cause
	for _, arg := range args {
		if err, ok := arg.(error); ok {
			cause = err
			break
		}
	}

	// 2. 渲染当前层 msg：%w 只用于标记 cause，渲染时按 %v 处理，
	// 参数列表保持完整，避免占位符与参数位置错位。
	msg := format
	causeRendered := false
	if len(args) > 0 && strings.Contains(format, "%") {
		msg = fmt.Sprintf(strings.ReplaceAll(format, "%w", "%v"), args...)
		causeRendered = cause != nil
	}

	file, line, fn := getCallerInfo(2)
	return &customError{
		msg:           msg,
		file:          file,
		line:          line,
		funcN:         fn,
		cause:         cause,
		causeRendered: causeRendered,
	}
}

// getCallerInfo 获取带有相对路径的文件信息
func getCallerInfo(skip int) (string, int, string) {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown", 0, "unknown"
	}
	fn := runtime.FuncForPC(pc).Name()
	if lastDot := strings.LastIndex(fn, "."); lastDot != -1 {
		fn = fn[lastDot+1:]
	}
	return file, line, fn
}

// FormatErrorFunc 用于快速封装error并将原error作为
func FormatErrorFunc(fmtErr error) func(origin error) error {
	return func(origin error) error {
		if origin == nil {
			return nil
		}

		// fmtErr 作为底层 cause，并保存堆栈链路为字符串
		return Wrap(fmtErr, fmt.Sprintf("%+v", origin))
	}
}
