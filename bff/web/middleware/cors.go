package middleware

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"strings"
	"time"
)

func NewCorsMiddleware() *CorsMiddleware {
	return &CorsMiddleware{}
}

type CorsMiddleware struct{}

func (c *CorsMiddleware) MiddlewareFunc() gin.HandlerFunc {
	return cors.New(cors.Config{
		// 允许的请求头
		AllowHeaders: []string{"Content-Type", "Authorization"},
		// 添加到响应头去,默认的响应头是不能够显示自定义的部分的
		ExposeHeaders: []string{"x-jwt-token", "x-refresh-token"},
		// 是否允许携带凭证（如 Cookies）
		AllowCredentials: true,
		// 解决跨域问题,当是以localhost或者bigdust.space开头的时候就允许跨域
		AllowOriginFunc: func(origin string) bool {
			//检测请求来源是否以localhost开头
			if strings.HasPrefix(origin, "localhost") {
				return true
			}
			//检测请求来源是否以bigdust.space开头 TODO 记得改成木犀的
			return strings.Contains(origin, "")
		},

		// 预检请求的缓存时间
		MaxAge: 12 * time.Hour,
	})
}
