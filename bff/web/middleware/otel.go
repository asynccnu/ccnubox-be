package middleware

import (
	"errors"

	"github.com/asynccnu/ccnubox-be/bff/errs"
	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// 给中间件注入依赖的框架
type OtelMiddleware struct{}

func NewOtelMiddlerware() *OtelMiddleware {
	return &OtelMiddleware{}
}

func (m *OtelMiddleware) Middleware() gin.HandlerFunc {
	return otelgin.Middleware("bff")
}

// 全局中间件，为链路的头 span 添加自定义 tag
// 如果没有学号的话那就不加
func (m *OtelMiddleware) AttributeMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Next()

		// 判断 ctx 中有没有用户信息
		_, exists := ctx.Get(ginx.UC_CTX)
		if !exists {
			return
		}

		// 如果有用户信息则进行更安全的学号信息读取
		uc, err := ginx.GetClaims[ijwt.UserClaims](ctx)
		if err != nil {
			ctx.Error(errs.UNAUTHORIED_ERROR(errors.New("链路获取学号失败")))
			return
		}

		span := trace.SpanFromContext(ctx)
		span.SetAttributes(attribute.String("student_id", uc.StudentId))
	}
}
