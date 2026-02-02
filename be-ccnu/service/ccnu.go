package service

import (
	"context"
	"crypto/rsa"
	"net/http"
	"time"

	proxyv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/proxy/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/tool"

	"github.com/asynccnu/ccnubox-be/be-ccnu/crawler"
	ccnuv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/ccnu/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

var (
	CCNUSERVER_ERROR       = errorx.FormatErrorFunc(ccnuv1.ErrorCcnuserverError("ccnu服务器错误"))
	Invalid_SidOrPwd_ERROR = errorx.FormatErrorFunc(ccnuv1.ErrorInvalidSidOrPwd("账号密码错误"))
	SYSTEM_ERROR           = errorx.FormatErrorFunc(ccnuv1.ErrorSystemError("系统内部错误"))
)

// 这里的err之所以在GetXKCookie和LoginCCNU两个方法里面不进行包装是因为如果进行封装了会导致error类型无法对应上kratos的error导致无法断言
func (c *ccnuService) GetXKCookie(ctx context.Context, studentId string, password string, tpe ...string) (string, error) {
	stuType := tool.ParseStudentType(studentId)
	switch stuType {
	case tool.UnderGraduate:
		cookie, err := c.getUnderGradCookie(ctx, studentId, password, tpe...)
		if err != nil {
			return "", err
		}
		return cookie, nil
	case tool.PostGraduate:
		cookie, err := c.getGradCookie(ctx, studentId, password)
		if err != nil {
			return "", err
		}
		return cookie, nil
	default:
		return "", Invalid_SidOrPwd_ERROR(errorx.New("studentId format invalid"))
	}
}

func (c *ccnuService) LoginCCNU(ctx context.Context, studentId string, password string) (bool, error) {
	tlog := c.l.WithContext(ctx)
	stuType := tool.ParseStudentType(studentId)

	switch stuType {
	case tool.PostGraduate:
		addr, err := c.GetProxyAddr(ctx)
		if err != nil {
			tlog.Warn("LoginCCNU GetProxyAddr err", logger.Error(err))
		}

		pg := crawler.NewPostGraduate(crawler.NewCrawlerClient(c.timeout, addr))
		ok, err := c.loginGrad(ctx, pg, studentId, password)
		if err != nil {
			return false, err
		}
		return ok, nil

	case tool.UnderGraduate:
		_, ok, err := c.loginUnderGrad(ctx, studentId, password)
		if err != nil {
			return false, err
		}
		return ok, nil

	default:
		return false, Invalid_SidOrPwd_ERROR(errorx.New("studentId format invalid"))
	}
}

func (c *ccnuService) loginGrad(ctx context.Context, pg *crawler.PostGraduate, studentId string, password string) (bool, error) {
	var isInCorrectPASSWORD = false

	pubkey, err := tool.Retry(func() (*rsa.PublicKey, error) {
		return pg.FetchPublicKey(ctx)
	})
	if err != nil {
		return false, CCNUSERVER_ERROR(errorx.Errorf("loginGrad FetchPublicKey error: %w", err))
	}

	_, err = tool.Retry(func() (string, error) {
		err := pg.LoginPostgraduateSystem(ctx, studentId, password, pubkey)
		if errorx.Is(err, crawler.INCorrectPASSWORD) {
			isInCorrectPASSWORD = true
			return "", nil
		}
		return "", err
	})

	if isInCorrectPASSWORD {
		return false, Invalid_SidOrPwd_ERROR(errorx.New("loginGrad incorrect password"))
	}
	if err != nil {
		return false, CCNUSERVER_ERROR(errorx.Errorf("loginGrad LoginPostgraduateSystem error: %w", err))
	}
	return true, nil
}

func (c *ccnuService) loginUnderGrad(ctx context.Context, studentId string, password string) (*http.Client, bool, error) {
	tlog := c.l.WithContext(ctx)
	addr, err := c.GetProxyAddr(ctx)
	if err != nil {
		tlog.Warn("loginUnderGrad GetProxyAddr err", logger.Error(err))
	}

	ps := crawler.NewPassport(crawler.NewCrawlerClient(c.timeout, addr))
	flag, err := ps.LoginPassport(ctx, studentId, password)
	if err != nil {
		if errorx.Is(err, crawler.INCorrectPASSWORD) {
			return nil, flag, Invalid_SidOrPwd_ERROR(errorx.Errorf("loginUnderGrad passport error: %w", err))
		}
		return nil, flag, CCNUSERVER_ERROR(errorx.Errorf("loginUnderGrad internal error: %w", err))
	}
	return ps.Client, flag, nil
}

func (c *ccnuService) getUnderGradCookie(ctx context.Context, stuId, password string, tpe ...string) (string, error) {
	tlog := c.l.WithContext(ctx)
	addr, err := c.GetProxyAddr(ctx)
	if err != nil {
		tlog.Warn("getUnderGradCookie GetProxyAddr err", logger.Error(err))
	}

	ug := crawler.NewUnderGrad(crawler.NewCrawlerClient(c.timeout, addr))
	client, ok, err := c.loginUnderGrad(ctx, stuId, password)
	if err != nil {
		return "", errorx.Errorf("getUnderGradCookie loginUnderGrad error: %w", err)
	}
	if !ok {
		// 如果登录没有报错但返回 flag 为 false，通常也是账号密码问题
		return "", Invalid_SidOrPwd_ERROR(errorx.New("getUnderGradCookie login failed"))
	}

	ug.Client = client
	_, err = tool.Retry(func() (string, error) {
		err := ug.LoginUnderGradSystem(ctx)
		if err != nil {
			return "", err
		}
		return "", nil
	})
	if err != nil {
		return "", CCNUSERVER_ERROR(errorx.Errorf("getUnderGradCookie LoginUnderGradSystem error: %w", err))
	}

	cookie, err := ug.GetCookieFromUnderGradSystem()
	if err != nil {
		return "", CCNUSERVER_ERROR(errorx.Errorf("getUnderGradCookie GetCookieFromUnderGradSystem error: %w", err))
	}

	return cookie, nil
}

func (c *ccnuService) getGradCookie(ctx context.Context, stuId, password string) (string, error) {
	tlog := c.l.WithContext(ctx)
	addr, err := c.GetProxyAddr(ctx)
	if err != nil {
		tlog.Warn("getGradCookie GetProxyAddr err", logger.Error(err))
	}

	pg := crawler.NewPostGraduate(crawler.NewCrawlerClient(c.timeout, addr))
	pubkey, err := tool.Retry(func() (*rsa.PublicKey, error) {
		return pg.FetchPublicKey(ctx)
	})
	if err != nil {
		return "", CCNUSERVER_ERROR(errorx.Errorf("getGradCookie FetchPublicKey error: %w", err))
	}

	cookie, err := pg.GetCookie(ctx, stuId, password, pubkey)
	if err != nil {
		return "", CCNUSERVER_ERROR(errorx.Errorf("getGradCookie GetCookie error: %w", err))
	}
	return cookie, nil
}

func (c *ccnuService) GetLibraryCookie(ctx context.Context, studentId, password string) (string, error) {
	l := crawler.NewLibrary(crawler.NewCrawlerClient(c.timeout, "")) // 这里简化了，实际可按需加 Proxy
	client, ok, err := c.loginUnderGrad(ctx, studentId, password)
	if err != nil {
		return "", errorx.Errorf("GetLibraryCookie loginUnderGrad error: %w", err)
	}
	if !ok {
		return "", Invalid_SidOrPwd_ERROR(errorx.New("GetLibraryCookie login failed"))
	}

	l.Client = client
	err = l.LoginLibrary(ctx)
	if err != nil {
		return "", CCNUSERVER_ERROR(errorx.Errorf("GetLibraryCookie LoginLibrary error: %w", err))
	}

	cookie, err := l.GetCookieFromLibrarySystem()
	if err != nil {
		return "", CCNUSERVER_ERROR(errorx.Errorf("GetLibraryCookie GetCookieFromLibrarySystem error: %w", err))
	}

	return cookie, nil
}

// TODO,许多逻辑用到,可以考虑抽象到pkg
func (c *ccnuService) GetProxyAddr(ctx context.Context) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	res, err := c.p.GetProxyAddr(cctx, &proxyv1.GetProxyAddrRequest{})
	if err != nil {
		return "", SYSTEM_ERROR(errorx.Errorf("GetProxyAddr rpc call error: %w", err))
	}

	return res.Addr, nil
}
