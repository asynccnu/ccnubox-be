package middleware

import (
	"github.com/asynccnu/ccnubox-be/bff/conf"
	"github.com/gin-gonic/gin"
)

type BasicAuthMiddleware struct {
	Username string
	Password string
}

func NewBasicAuthMiddleware(conf *conf.ServerConf) *BasicAuthMiddleware {

	return &BasicAuthMiddleware{
		Username: conf.BasicAuth.Username,
		Password: conf.BasicAuth.Password,
	}
}

func (m *BasicAuthMiddleware) MiddlewareFunc() gin.HandlerFunc {
	return gin.BasicAuth(gin.Accounts{m.Username: m.Password})
}
