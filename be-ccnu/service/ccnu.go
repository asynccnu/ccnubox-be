package service

import (
	"context"

	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/tool"

	ccnuv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/ccnu/v1"
)

var (
	CCNUSERVER_ERROR                           = tool.FormatCCNUErrorFunc(ccnuv1.ErrorCcnuserverError("ccnu服务器错误"), ccnuv1.ErrorCcnuserverError(tool.CCNUAccountInitializationRequiredMarker))
	CCNU_ACCOUNT_INITIALIZATION_REQUIRED_ERROR = errorx.FormatErrorFunc(ccnuv1.ErrorCcnuserverError(tool.CCNUAccountInitializationRequiredMarker))
	Invalid_SidOrPwd_ERROR                     = errorx.FormatErrorFunc(ccnuv1.ErrorInvalidSidOrPwd("账号密码错误"))
	SYSTEM_ERROR                               = errorx.FormatErrorFunc(ccnuv1.ErrorSystemError("系统内部错误"))
)

// 这里的err之所以在GetXKCookie和LoginCCNU两个方法里面不进行包装是因为如果进行封装了会导致error类型无法对应上kratos的error导致无法断言
func (c *ccnuService) GetXKCookie(ctx context.Context, studentId string, password string, tpe ...string) (string, error) {
	cr := c.factory.New()
	stuType := tool.ParseStudentType(studentId)
	switch stuType {
	case tool.UnderGraduate:
		cookie, err := cr.GetUnderGraduateCookie(ctx, studentId, password, tpe...)
		return cookie, mapCrawlerError(err)
	case tool.PostGraduate:
		cookie, err := cr.GetPostGraduateCookie(ctx, studentId, password)
		return cookie, mapCrawlerError(err)
	default:
		return "", Invalid_SidOrPwd_ERROR(errorx.New("studentId format invalid"))
	}
}

func (c *ccnuService) LoginCCNU(ctx context.Context, studentId string, password string) (bool, error) {
	cr := c.factory.New()
	stuType := tool.ParseStudentType(studentId)

	switch stuType {
	case tool.PostGraduate:
		ok, err := cr.LoginPostGraduate(ctx, studentId, password)
		return ok, mapCrawlerError(err)

	case tool.UnderGraduate:
		ok, err := cr.LoginUnderGraduate(ctx, studentId, password)
		return ok, mapCrawlerError(err)

	default:
		return false, Invalid_SidOrPwd_ERROR(errorx.New("studentId format invalid"))
	}
}

func (c *ccnuService) GetLibraryToken(ctx context.Context, studentId, password string, service ccnuv1.LIBRARY_TYPE) (string, error) {
	cr := c.factory.New()
	var (
		token string
		err   error
	)
	switch service {
	case ccnuv1.LIBRARY_TYPE_LIBRARY_SEAT:
		token, err = cr.GetLibrarySeatToken(ctx, studentId, password)
	case ccnuv1.LIBRARY_TYPE_LIBRARY_DISCUSSION:
		token, err = cr.GetLibraryDiscussionToken(ctx, studentId, password)
	}
	return token, mapCrawlerError(err)
}

func (c *ccnuService) CheckLibraryToken(ctx context.Context, token string, service ccnuv1.LIBRARY_TYPE) (bool, error) {
	cr := c.factory.New()
	var (
		ok  bool
		err error
	)
	switch service {
	case ccnuv1.LIBRARY_TYPE_LIBRARY_SEAT:
		ok, err = cr.CheckLibrarySeatToken(ctx, token)
	case ccnuv1.LIBRARY_TYPE_LIBRARY_DISCUSSION:
		ok, err = cr.CheckLibraryDiscussionToken(ctx, token)
	}
	return ok, mapCrawlerError(err)
}

func mapCrawlerError(err error) error {
	if err == nil {
		return nil
	}

	var crawlerErr *CrawlerError
	if errorx.As(err, &crawlerErr) {
		switch crawlerErr.Kind {
		case CrawlerInvalidCredential:
			return Invalid_SidOrPwd_ERROR(errorx.Errorf("crawler invalid credential: %w", err))
		case CrawlerAccountInitializationRequired:
			return CCNU_ACCOUNT_INITIALIZATION_REQUIRED_ERROR(errorx.Errorf("crawler account initialization required: %w", err))
		}
	}

	return CCNUSERVER_ERROR(errorx.Errorf("crawler upstream error: %w", err))
}
