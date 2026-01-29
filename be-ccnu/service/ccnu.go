package service

import (
	"context"
	"crypto/rsa"
	"errors"
	"net/http"
	"time"

	proxyv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/proxy/v1"
	"github.com/asynccnu/ccnubox-be/common/tool"

	"github.com/asynccnu/ccnubox-be/be-ccnu/crawler"
	ccnuv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/ccnu/v1"
	errorx "github.com/asynccnu/ccnubox-be/common/pkg/errorx/rpcerr"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

// 定义错误,这里将kratos的error作为一个重要部分传入,此处的错误并不直接在service中去捕获,而是选择在更底层的爬虫去捕获,因为爬虫的错误处理非常复杂
var (
	CCNUSERVER_ERROR = func(err error) error {
		return errorx.New(ccnuv1.ErrorCcnuserverError("ccnu服务器错误"), "ccnuServer", err)
	}

	Invalid_SidOrPwd_ERROR = func(err error) error {
		return errorx.New(ccnuv1.ErrorInvalidSidOrPwd("账号密码错误"), "user", err)
	}

	SYSTEM_ERROR = func(err error) error {
		return errorx.New(ccnuv1.ErrorSystemError("系统内部错误"), "system", err)
	}
)

func (c *ccnuService) GetXKCookie(ctx context.Context, studentId string, password string, tpe ...string) (string, error) {

	if len(studentId) > 4 && (studentId[4] == '1' || studentId[4] == '0') {
		// 研究生
		return c.getGradCookie(ctx, studentId, password)
	} else if len(studentId) > 4 && studentId[4] == '2' {
		//本科生
		return c.getUnderGradCookie(ctx, studentId, password, tpe...)
	} else {
		return "", Invalid_SidOrPwd_ERROR(errors.New("账号密码错误"))
	}
}

func (c *ccnuService) LoginCCNU(ctx context.Context, studentId string, password string) (bool, error) {
	tlog := c.l.WithContext(ctx)
	if len(studentId) > 4 && (studentId[4] == '1' || studentId[4] == '0') {
		addr, err := c.GetProxyAddr(ctx)
		if err != nil {
			tlog.Warn("LoginCCNU GetProxyAddr err", logger.Error(err))
		}

		// 研究生
		pg := crawler.NewPostGraduate(crawler.NewCrawlerClient(c.timeout, addr))
		return c.loginGrad(ctx, pg, studentId, password)
	} else if len(studentId) > 4 && studentId[4] == '2' {
		//本科生
		_, ok, err := c.loginUnderGrad(ctx, studentId, password)
		return ok, err
	} else {
		return false, Invalid_SidOrPwd_ERROR(errors.New("账号密码错误"))
	}

}

func (c *ccnuService) loginGrad(ctx context.Context, pg *crawler.PostGraduate, studentId string, password string) (bool, error) {
	var (
		isInCorrectPASSWORD = false //用于判断是否是账号密码错误
	)
	pubkey, err := tool.Retry(func() (*rsa.PublicKey, error) {
		return pg.FetchPublicKey(ctx)
	})
	if err != nil {
		return false, err
	}

	_, err = tool.Retry(func() (string, error) {
		err := pg.LoginPostgraduateSystem(ctx, studentId, password, pubkey)
		if errors.Is(err, crawler.INCorrectPASSWORD) {
			// 标识账号密码错误,强制结束
			isInCorrectPASSWORD = true
			return "", nil
		}
		return "", err
	})
	//如果密码有误
	if isInCorrectPASSWORD {
		return false, Invalid_SidOrPwd_ERROR(errors.New("账号密码错误"))
	}
	//如果存在错误
	if err != nil {
		return false, err
	}
	return true, nil
}

func (c *ccnuService) loginUnderGrad(ctx context.Context, studentId string, password string) (*http.Client, bool, error) {
	tlog := c.l.WithContext(ctx)
	addr, err := c.GetProxyAddr(ctx)
	if err != nil {
		tlog.Warn("loginUnderGrad GetProxyAddr err", logger.Error(err))
	}

	var (
		ps = crawler.NewPassport(crawler.NewCrawlerClient(c.timeout, addr))
	)

	flag, err := ps.LoginPassport(ctx, studentId, password)
	if errors.Is(err, crawler.INCorrectPASSWORD) {
		return nil, flag, Invalid_SidOrPwd_ERROR(err)
	}
	return ps.Client, flag, err
}

func (c *ccnuService) getUnderGradCookie(ctx context.Context, stuId, password string, tpe ...string) (string, error) {
	tlog := c.l.WithContext(ctx)
	addr, err := c.GetProxyAddr(ctx)
	if err != nil {
		tlog.Warn("getUnderGradCookie GetProxyAddr err", logger.Error(err))
	}

	//初始化client
	var (
		ug = crawler.NewUnderGrad(crawler.NewCrawlerClient(c.timeout, addr))
	)

	client, ok, err := c.loginUnderGrad(ctx, stuId, password)
	if err != nil || !ok {
		return "", err
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
		return "", err
	}

	cookie, err := ug.GetCookieFromUnderGradSystem()
	if err != nil {
		return "", err
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
		return "", err
	}
	return pg.GetCookie(ctx, stuId, password, pubkey)
}

func (c *ccnuService) GetLibraryCookie(ctx context.Context, studentId, password string) (string, error) {
	tlog := c.l.WithContext(ctx)
	addr, err := c.GetProxyAddr(ctx)
	if err != nil {
		tlog.Warn("GetLibraryCookie GetProxyAddr err", logger.Error(err))
	}

	// 初始化Client
	var (
		l = crawler.NewLibrary(crawler.NewCrawlerClient(c.timeout, addr))
	)

	client, ok, err := c.loginUnderGrad(ctx, studentId, password)
	if err != nil || !ok {
		return "", err
	}

	l.Client = client

	err = l.LoginLibrary(ctx)
	if err != nil {
		return "", err
	}

	cookie, err := l.GetCookieFromLibrarySystem()
	if err != nil {
		return "", err
	}

	return cookie, nil
}

func (c *ccnuService) GetProxyAddr(ctx context.Context) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	res, err := c.p.GetProxyAddr(cctx, &proxyv1.GetProxyAddrRequest{})
	if err != nil {
		return "", err
	}

	return res.Addr, nil
}
