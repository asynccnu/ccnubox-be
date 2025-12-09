// 由于 rpc 层和 api 层的错误处理机制不一样
// 这里拆成两个子包来作为公共包
package rpcerr

import (
	"errors"
	"fmt"

	"github.com/asynccnu/ccnubox-be/be-pkg/errorx"
)

// CustomError 自定义错误类型,相比一般的错误减少了code部分,code完全让bff来控制
// 内部系统服务选择放弃code,如果需要使用状态控制可以选择使用grpc的status字段
type CustomError struct {
	ERR error // 暴露给调用方的错误信息
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
		return fmt.Sprintf("type:%s %s (at %s:%d in %s): %v", e.Category, e.ERR.Error(), e.File, e.Line, e.Function, e.Cause)
	}
	return fmt.Sprintf("type:%s %s (at %s:%d in %s)", e.Category, e.ERR.Error(), e.File, e.Line, e.Function)
}

// New 创建新的 CustomError
func New(ERR error, category string, cause error) error {
	// 获取调用栈信息
	file, line, function := errorx.GetCallerInfo(3)
	return &CustomError{
		ERR:      ERR,
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
