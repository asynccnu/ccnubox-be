package tool

import (
	"errors"
	"strings"

	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
)

const CCNUAccountInitializationRequiredMarker = "CCNU_ACCOUNT_INITIALIZATION_REQUIRED"

var ErrCCNUAccountInitializationRequired = errors.New(CCNUAccountInitializationRequiredMarker)

// FormatCCNUErrorFunc 在固定业务错误包装中保留初始化分类，避免 RPC 解包后丢失标记。
// 两个业务错误均使用固定公开消息，原始错误只保留在服务内日志链路中。
func FormatCCNUErrorFunc(normal, initialization error) func(error) error {
	normalWrap := errorx.FormatErrorFunc(normal)
	initializationWrap := errorx.FormatErrorFunc(initialization)
	return func(err error) error {
		if IsCCNUAccountInitializationRequired(err) {
			return initializationWrap(err)
		}
		return normalWrap(err)
	}
}

func IsCCNUAccountInitializationRequired(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrCCNUAccountInitializationRequired) ||
		strings.Contains(err.Error(), CCNUAccountInitializationRequiredMarker)
}
