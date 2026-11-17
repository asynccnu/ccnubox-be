package errorx

import (
	"errors"
	"fmt"
	"io"
	"runtime"
	"strings"
	"unicode/utf8"
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
	causeIdx := -1
	// 1. 找到第一个 error 类型的参数作为 cause
	for i, arg := range args {
		if err, ok := arg.(error); ok {
			cause = err
			causeIdx = i
			break
		}
	}

	// 2. 渲染当前层 msg：%w 只用于标记 cause，渲染时按 %v 处理，
	// 参数列表保持完整，避免占位符与参数位置错位。
	msg := format
	causeRendered := false
	if len(args) > 0 && strings.Contains(format, "%") {
		scan := scanFormat(format)
		msg = fmt.Sprintf(scan.format, args...)
		// 只有 cause 真的被某个占位符渲染进 msg 时，Error() 才不再拼接它。
		causeRendered = cause != nil && scan.consumes(causeIdx)
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

// formatScan 是格式串的扫描结果。
type formatScan struct {
	// format 把 %w 动词替换成 %v 后的格式串，交给 fmt.Sprintf 渲染。
	format string
	// consumed 记录被占位符消费的参数下标（0 起，含 * 宽度/精度与 %[n] 显式索引）。
	consumed map[int]struct{}
}

func (f formatScan) consumes(idx int) bool {
	_, ok := f.consumed[idx]
	return ok
}

// scanFormat 按 fmt 的占位符语法扫描格式串：只替换真正的 %w 动词，跳过 %% 字面量，
// 并记录被消费的参数下标。
func scanFormat(format string) formatScan {
	scan := formatScan{format: format, consumed: make(map[int]struct{})}
	var b strings.Builder
	b.Grow(len(format))
	next := 0
	for i := 0; i < len(format); {
		if format[i] != '%' {
			b.WriteByte(format[i])
			i++
			continue
		}
		if i+1 < len(format) && format[i+1] == '%' {
			// %% 是字面量，既不消费参数也不改写后续字符。
			b.WriteString("%%")
			i += 2
			continue
		}
		start := i
		i++
		// flags
		for i < len(format) && strings.IndexByte("+-# 0", format[i]) >= 0 {
			i++
		}
		// 宽度：可带显式索引，或由 * 消费一个参数
		next, i = scanArgIndex(format, i, next)
		if i < len(format) && format[i] == '*' {
			scan.markConsumed(next)
			next++
			i++
		} else {
			i = scanDigits(format, i)
		}
		// 精度
		if i < len(format) && format[i] == '.' {
			i++
			next, i = scanArgIndex(format, i, next)
			if i < len(format) && format[i] == '*' {
				scan.markConsumed(next)
				next++
				i++
			} else {
				i = scanDigits(format, i)
			}
		}
		// 动词前的显式索引
		next, i = scanArgIndex(format, i, next)
		if i >= len(format) {
			// 结尾的孤立 %，原样交给 fmt 处理
			b.WriteString(format[start:])
			break
		}
		verb, size := utf8.DecodeRuneInString(format[i:])
		b.WriteString(format[start:i])
		if verb == 'w' {
			b.WriteByte('v')
		} else {
			b.WriteString(format[i : i+size])
		}
		scan.markConsumed(next)
		next++
		i += size
	}
	scan.format = b.String()
	return scan
}

// scanArgIndex 解析占位符中的 %[n] 显式参数索引，返回更新后的下一参数下标与新的扫描位置。
func scanArgIndex(format string, i, next int) (int, int) {
	if i >= len(format) || format[i] != '[' {
		return next, i
	}
	j := i + 1
	n := 0
	for j < len(format) && format[j] >= '0' && format[j] <= '9' {
		n = n*10 + int(format[j]-'0')
		j++
	}
	if j >= len(format) || format[j] != ']' {
		return next, i
	}
	if n > 0 {
		return n - 1, j + 1
	}
	// [0] 不是合法索引，fmt 会跳过它，这里同样不改变参数下标。
	return next, j + 1
}

func scanDigits(format string, i int) int {
	for i < len(format) && format[i] >= '0' && format[i] <= '9' {
		i++
	}
	return i
}

func (f formatScan) markConsumed(idx int) {
	if idx < 0 {
		return
	}
	f.consumed[idx] = struct{}{}
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
