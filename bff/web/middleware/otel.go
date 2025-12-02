package middleware

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

type OtelMiddleware struct {
	middleware gin.HandlerFunc
}

func NewOtelMiddlerware() *OtelMiddleware {
	return &OtelMiddleware{
		middleware: otelgin.Middleware("bff"),
	}
}

func (m *OtelMiddleware) Middleware() gin.HandlerFunc {
	return m.middleware
}
