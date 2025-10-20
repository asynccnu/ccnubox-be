package service

import (
	"context"
	"crypto/rsa"
	"errors"
	ccnuv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/ccnu/v1"
	"github.com/asynccnu/ccnubox-be/be-ccnu/crawler"
	"github.com/asynccnu/ccnubox-be/be-ccnu/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/be-ccnu/tool"
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

func (c *ccnuService) GetXKCookie(ctx context.Context, studentId string, password string) (string, error) {

	if len(studentId) > 4 && (studentId[4] == '1' || studentId[4] == '0') {
		// 研究生
		return c.getGradCookie(ctx, studentId, password)
	} else if len(studentId) > 4 && studentId[4] == '2' {
		//本科生
		return c.getUnderGradCookie(ctx, studentId, password)
	} else {
		return "", Invalid_SidOrPwd_ERROR(errors.New("账号密码错误"))
	}
}

func (c *ccnuService) LoginCCNU(ctx context.Context, studentId string, password string) (bool, error) {
	if len(studentId) > 4 && (studentId[4] == '1' || studentId[4] == '0') {
		// 研究生
		pg := crawler.NewPostGraduate(crawler.NewCrawlerClient(c.timeout))
		return c.loginGrad(ctx, pg, studentId, password)
	} else if len(studentId) > 4 && studentId[4] == '2' {
		//本科生
		ug := crawler.NewUnderGrad(crawler.NewCrawlerClient(c.timeout))
		return c.loginUnderGrad(ctx, ug, studentId, password)
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

func (c *ccnuService) loginUnderGrad(ctx context.Context, ug *crawler.UnderGrad, studentId string, password string) (bool, error) {
	var (
		isInCorrectPASSWORD = false //用于判断是否是账号密码错误
	)

	params, err := tool.Retry(func() (*crawler.AccountRequestParams, error) {
		return ug.GetParamsFromHtml(ctx)
	})
	if err != nil {
		return false, err
	}

	//此处比较特殊由于账号密码错误是必然无效的请求,应当直接返回
	_, err = tool.Retry(func() (string, error) {
		err := ug.LoginCCNUPassport(ctx, studentId, password, params)
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

func (c *ccnuService) getUnderGradCookie(ctx context.Context, stuId, password string) (string, error) {
	//初始化client
	var (
		ug = crawler.NewUnderGrad(crawler.NewCrawlerClient(c.timeout))
	)

	ok, err := c.loginUnderGrad(ctx, ug, stuId, password)
	if err != nil || !ok {
		return "", err
	}

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
	pg := crawler.NewPostGraduate(crawler.NewCrawlerClient(c.timeout))
	pubkey, err := tool.Retry(func() (*rsa.PublicKey, error) {
		return pg.FetchPublicKey(ctx)
	})
	if err != nil {
		return "", err
	}
	return pg.GetCookie(ctx, stuId, password, pubkey)
}
