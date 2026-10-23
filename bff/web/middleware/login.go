package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"

	"github.com/asynccnu/ccnubox-be/bff/errs"
	b_errorx "github.com/asynccnu/ccnubox-be/bff/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/service/activity"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

type LoginMiddleware struct {
	handler ijwt.Handler
	redis   redis.Cmdable
	now     func() time.Time
}

func NewLoginMiddleWare(hdl ijwt.Handler, r redis.Cmdable) *LoginMiddleware {
	return &LoginMiddleware{handler: hdl, redis: r, now: time.Now}
}

func (m *LoginMiddleware) MiddlewareFunc() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		uc, err := m.extractUserClaimsFromAuthorizationHeader(ctx)
		if err != nil {
			ctx.Error(errs.UNAUTHORIZED_ERROR(errorx.Errorf("身份验证失败: %w", err)))
			return
		}
		// Persist the activity intent before exposing authenticated claims or
		// running handlers. A failed admission must not become untracked traffic.
		if uc.StudentId != "" {
			recordCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			err := activity.Record(recordCtx, m.redis, uc.StudentId, m.now())
			cancel()
			if err != nil {
				const message = "活跃明细登记暂时不可用，请稍后重试"
				ctx.Error(errors.Join(b_errorx.New(http.StatusServiceUnavailable, errs.INTERNAL_SERVER_ERROR_CODE, message), err))
				ctx.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"code": errs.INTERNAL_SERVER_ERROR_CODE, "msg": message, "data": nil})
				return
			}
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
		return ijwt.UserClaims{}, errorx.New("authorization为空")
	}
	// Bearer xxxx
	segs := strings.Split(authCode, " ")

	if len(segs) != 2 {
		return ijwt.UserClaims{}, errorx.New("authorization为空格式不合理")
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
		return ijwt.UserClaims{}, errorx.New("token无效")
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
		return ijwt.UserClaims{}, errorx.New("session检验：失败")
	}
	return uc, nil
}
