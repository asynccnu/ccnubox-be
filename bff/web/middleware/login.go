package middleware

import (
	"errors"
	userv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/user/v1"
	"strings"

	"github.com/asynccnu/ccnubox-be/bff/errs"
	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type LoginMiddleware struct {
	handler    ijwt.Handler
	userClient userv1.UserServiceClient
}

func NewLoginMiddleWare(hdl ijwt.Handler, userClient userv1.UserServiceClient) *LoginMiddleware {

	l := &LoginMiddleware{
		handler:    hdl,
		userClient: userClient,
	}
	return l
}

func (m *LoginMiddleware) MiddlewareFunc() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		uc, err := m.extractUserClaimsFromAuthorizationHeader(ctx)
		if err != nil {
			ctx.Error(errs.UNAUTHORIED_ERROR(errors.New("身份验证失败!")))
			return
		}
		// 设置claims并执行下一个
		ginx.SetClaims[ijwt.UserClaims](ctx, uc)
		ctx.Next()
	}
}

func (m *LoginMiddleware) extractUserClaimsFromAuthorizationHeader(ctx *gin.Context) (ijwt.UserClaims, error) {
	authCode := ctx.GetHeader("Authorization")

	// 没token
	if authCode == "" {
		return ijwt.UserClaims{}, errors.New("authorization为空")
	}
	// Bearer xxxx
	segs := strings.Split(authCode, " ")

	if len(segs) != 2 {
		return ijwt.UserClaims{}, errors.New("authorization为空格式不合理")
	}

	tokenStr := segs[1]
	uc := ijwt.UserClaims{}

	token, err := jwt.ParseWithClaims(tokenStr, &uc, func(*jwt.Token) (interface{}, error) {
		// 可以根据具体情况给出不同的key
		return m.handler.JWTKey(), nil
	})
	if err != nil {
		return ijwt.UserClaims{}, err
	}

	if token == nil || !token.Valid {
		return ijwt.UserClaims{}, errors.New("token无效")
	}

	// token有效
	// User-Agent
	//if uc.UserAgent != ctx.GetHeader("User-Agent") {
	//	// 大概率是攻击者才会进入这个分支
	//	return ijwt.UserClaims{}, errors.New("User-Agent验证：不安全")
	//}

	ok, err := m.handler.CheckSession(ctx, uc.Ssid)
	if err != nil || ok {
		// err如果是redis崩溃导致，考虑进行降级，不再验证是否退出 refresh_token降级的话收益会很少，因为是低频接口
		return ijwt.UserClaims{}, errors.New("session检验：失败")
	}
	password, err := m.handler.DecryptPasswordFromClaims(&uc)
	if err != nil {
		return ijwt.UserClaims{}, err
	}
	// TODO 临时逻辑,用于解决秘钥不统一的问题,后续需要删除
	_, err = m.userClient.SaveUser(ctx, &userv1.SaveUserReq{
		StudentId: uc.StudentId,
		Password:  password,
	})
	if err != nil {
		return ijwt.UserClaims{}, err
	}

	return uc, nil
}
