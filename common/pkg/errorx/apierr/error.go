// 由于 rpc 层和 api 层的错误处理机制不一样
// 这里拆成两个子包来作为公共包
package apierr

import (
	"errors"
	"fmt"

	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
)

// CustomError 自定义错误类型
type CustomError struct {
	HttpCode int    // http错误
	Code     int    // 具体错误码
	Msg      string // 暴露给前端的错误信息
	//内部日志
	Category string //具体分类
	Cause    error  // 具体错误原因
	File     string // 出错的文件名
	Line     int    // 出错的行号
	Function string // 出错的函数名
}

// Error 实现 errorx 接口
func (e *CustomError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("type:%s [%d] %s (at %s:%d in %s): %v", e.Category, e.Code, e.Msg, e.File, e.Line, e.Function, e.Cause)
	}
	return fmt.Sprintf("type:%s [%d] %s (at %s:%d in %s)", e.Category, e.Code, e.Msg, e.File, e.Line, e.Function)
}

// New 创建新的 CustomError
func New(httpCode int, code int, message string, category string, cause error) error {
	// 获取调用栈信息
	file, line, function := errorx.GetCallerInfo(3)
	return &CustomError{
		HttpCode: httpCode,
		Code:     code,
		Msg:      message,
		Category: category,
		Cause:    cause,
		File:     file,
		Line:     line,
		Function: function,
	}
}

// 转换为自定义错误类型
func ToCustomError(err error) *CustomError {
	var customErr *CustomError
	if errors.As(err, &customErr) {
		return customErr
	} else {
		return nil
	}
}
