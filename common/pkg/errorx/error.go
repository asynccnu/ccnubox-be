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
}

// Error 保持简洁：只负责返回单行错误链
// 效果：层级1: 层级2: 底层错误
func (e *customError) Error() string {
	if e.cause == nil {
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
				fmt.Fprintf(s, "\n%+v", e.cause)
			}
			return
		}
		// 普通 %v
		io.WriteString(s, e.Error())
	case 's':
		io.WriteString(s, e.Error())
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

// Errorf 格式化并识别 %w
// 优化：msg 字段只保留当前层的格式化结果，不合并子错误文本
func Errorf(format string, args ...any) error {
	var cause error
	var causeIdx = -1

	// 1. 找到第一个 error 类型的参数作为 cause，并记录它的下标
	for i, arg := range args {
		if err, ok := arg.(error); ok {
			cause = err
			causeIdx = i
			break
		}
	}

	// 2. 准备渲染当前层 msg 的参数
	// 我们需要从 args 中剔除掉那个被当作 cause 的 error，否则 Sprintf 会多出一个参数
	renderArgs := make([]any, 0)
	for i, arg := range args {
		if i == causeIdx {
			continue // 跳过 cause
		}
		renderArgs = append(renderArgs, arg)
	}

	// 3. 处理 format 字符串，去掉 %w 或 %v 占位符
	// 我们只渲染当前层的逻辑描述
	msgFormat := format
	if idx := strings.Index(format, "%w"); idx != -1 {
		msgFormat = strings.TrimSuffix(format[:idx], ": ")
	} else if cause != nil {
		// 如果没有 %w 但有 error，且 format 里有 %v，通常最后一个 %v 是给 error 的
		if idx := strings.LastIndex(format, "%v"); idx != -1 {
			msgFormat = strings.TrimSuffix(format[:idx], ": ")
		}
	}

	// 4. 渲染当前层的 msg
	msg := msgFormat
	// 捕获可能由于占位符不匹配导致的 panic 或错误格式
	if strings.Contains(msgFormat, "%") && len(renderArgs) > 0 {
		msg = fmt.Sprintf(msgFormat, renderArgs...)
	}

	file, line, fn := getCallerInfo(2)
	return &customError{
		msg:   msg,
		file:  file,
		line:  line,
		funcN: fn,
		cause: cause,
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
