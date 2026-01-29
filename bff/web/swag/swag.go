package swag

import (
	"os"
	"os/exec"

	"github.com/asynccnu/ccnubox-be/bff/errs"
	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/web"
	"github.com/gin-gonic/gin"
)

type SwagHandler struct{}

func NewSwagHandler() *SwagHandler {
	return &SwagHandler{}
}

func (c *SwagHandler) RegisterRoutes(s *gin.RouterGroup, basicAuthMiddleware gin.HandlerFunc) {
	s.GET("/swag", basicAuthMiddleware, ginx.Wrap(c.GetOpenApi3))
}

// GetOpenApi3 获取/重新生成 OpenAPI3 接口文档
// @Summary 获取 OpenAPI3 接口文档 (YAML)
// @Description 接口直接返回 docs/openapi3.yaml yaml格式的原始内容，使用BasicAuth进行验证
// @Tags swag
// @Produce application/x-yaml
// @Security BasicAuth
// @Success 200 {string} string
// @Router /swag [get]
func (c *SwagHandler) GetOpenApi3(ctx *gin.Context) (web.Response, error) {
	// 每次访问该接口直接重新生成并获取swag实现开发端和运行端获取的接口文档跟实际使用代码相同
	cmd := exec.Command("make", "swag")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return web.Response{}, errs.MAKE_SWAG_ERROR(err)
	}

	filepath := "docs/openapi3.yaml"
	content, err := os.ReadFile(filepath)
	if err != nil {
		return web.Response{}, errs.OPEN_SWAG_ERROR(err)
	}

	// 返回 YAML 字符串
	ctx.String(200, string(content))
	// 为保证返回的文件纯净性，不打印通用响应体
	ctx.Abort()
	return web.Response{}, nil
}
