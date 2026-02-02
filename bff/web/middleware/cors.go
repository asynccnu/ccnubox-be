package middleware

import (
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
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
		ExposeHeaders: []string{"x-jwt-token", "x-refresh-token", "X-Trace-Id"}, //前面那两个不太敢动,正常格式是要大小写对齐的,等之后有了测试环境再试
		// 是否允许携带凭证（如 Cookies）
		AllowCredentials: true,
		// 解决跨域问题,当是以localhost或者muxixyz.com开头的时候就允许跨域
		AllowOriginFunc: func(origin string) bool {
			//检测请求来源是否以localhost开头
			if strings.HasPrefix(origin, "localhost") {
				return true
			}
			//检测请求来源是否以muxixyz.com开头 TODO 这里目前没有限制可能需要修改
			return strings.Contains(origin, "")
		},

		// 预检请求的缓存时间
		MaxAge: 12 * time.Hour,
	})
}
