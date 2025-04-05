package ginx

import (
	"errors"
	api_errors "github.com/asynccnu/ccnubox-be/bff/errs"
	"github.com/gin-gonic/gin"
	"net/http"
)

const RESP_CTX = "ginx_resp"
const UC_CTX = "ginx_user"

// ctx表示上下文,req表示请求结构体,Resp表示响应结构体(这里全部填web.Response),UserClaims表示用户信息
func WrapClaimsAndReq[Req any, UserClaims any, Resp any](fn func(*gin.Context, Req, UserClaims) (Resp, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {

		//检查前置中间件是否存在错误,如果存在应当直接返回
		if len(ctx.Errors) > 0 {
			return
		}
		//解析请求
		var req Req
		err := bind(ctx, &req)
		if err != nil {
			ctx.Error(err)
			return
		}

		//获取uc参数
		uc, err := GetClaims[UserClaims](ctx)
		if err != nil {
			ctx.Error(err)
			return
		}

		//执行函数
		res, err := fn(ctx, req, uc)
		if err != nil {
			ctx.Error(err)
		} else {
			ctx.Set(RESP_CTX, res)
		}

	}
}

// WrapReq 。用于处理有请求体的请求
// ctx表示上下文,req表示请求结构体,Resp表示响应结构体(这里全部填web.Response)
func WrapReq[Req any, Resp any](fn func(*gin.Context, Req) (Resp, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		//检查前置中间件是否存在错误,如果存在应当直接返回
		if len(ctx.Errors) > 0 {
			return
		}
		//解析参数
		var req Req
		err := bind(ctx, &req)
		if err != nil {
			ctx.Error(err)
			return
		}

		// 调用业务逻辑函数
		res, err := fn(ctx, req)
		if err != nil {
			ctx.Error(err)
			return
		} else {
			ctx.Set(RESP_CTX, res)
		}
	}
}

// Wrap 。用于处理没有请求体的请求
// ctx表示上下文,Resp表示响应结构体(这里全部填web.Response)
func Wrap[Resp any](fn func(*gin.Context) (Resp, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		//检查前置中间件是否存在错误,如果存在应当直接返回
		if len(ctx.Errors) > 0 {
			return
		}
		res, err := fn(ctx)
		if err != nil {
			ctx.Error(err)
			return
		} else {
			ctx.Set(RESP_CTX, res)
		}
	}
}

// WrapClaims 用于处理有用户验证但是没有请求体的请求
// ctx表示上下文,Resp表示响应结构体(这里全部填web.Response),UserClaims表示用户信息
func WrapClaims[UserClaims any, Resp any](fn func(*gin.Context, UserClaims) (Resp, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		//检查前置中间件是否存在错误,如果存在应当直接返回
		if len(ctx.Errors) > 0 {
			return
		}
		//获取uc参数
		uc, err := GetClaims[UserClaims](ctx)
		if err != nil {
			ctx.Error(err)
			return
		}
		//执行函数
		res, err := fn(ctx, uc)
		if err != nil {
			ctx.Error(err)
			return
		} else {
			ctx.Set(RESP_CTX, res)
		}
	}
}

// 解析参数通用函数
func bind(ctx *gin.Context, req any) error {
	var err error
	// 根据请求方法选择合适的绑定方式
	if ctx.Request.Method == http.MethodGet {
		err = ctx.ShouldBindQuery(req) // 处理GET请求的查询参数
	} else {
		err = ctx.ShouldBind(req) // 处理POST、PUT等请求的请求体数据
	}

	if err != nil {
		return api_errors.BAD_ENTITY_ERROR(err)
	}

	return nil
}

// 获取Claims通用函数
func GetClaims[UserClaims any](ctx *gin.Context) (claims UserClaims, err error) {
	rawVal, ok := ctx.Get(UC_CTX)
	if !ok {
		return claims, api_errors.BAD_ENTITY_ERROR(errors.New("从上下文获取userClaims失败"))
	}

	// 注意，这里要求放进去 ctx 的不能是*UserClaims，这是常见的一个错误
	claims, ok = rawVal.(UserClaims)
	if !ok {
		return claims, api_errors.BAD_ENTITY_ERROR(errors.New("userClaims类型断言失败了"))
	}

	return claims, nil
}

// 设置Claims通用函数
func SetClaims[UserClaims any](ctx *gin.Context, claims UserClaims) {
	ctx.Set(UC_CTX, claims)
}

func GetResp[Resp any](ctx *gin.Context) Resp {
	var resp Resp
	respVal, ok := ctx.Get(RESP_CTX)
	if ok {
		if resp, ok = respVal.(Resp); ok {
			return resp
		}
	}
	return resp
}
