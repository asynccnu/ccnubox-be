package errcode

import (
	v1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/library/v1"
	"github.com/go-kratos/kratos/v2/errors"
)

var (
	ErrCrawler   = errors.New(456, v1.ErrorReason_Crawler_Error.String(), "爬取课表失败")
	ErrCCNULogin = errors.New(457, v1.ErrorReason_CCNULogin_Error.String(), "请求ccnu一站式登录服务错误")
)
