package content

import (
	"github.com/asynccnu/ccnubox-be/bff/errs"
	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/web"
	contentv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/content/v1"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
)

func (h *ContentHandler) RegisterUpdateVersionRoute(group *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	sg := group.Group("/version")
	sg.GET("/getVersion", ginx.Wrap(h.GetUpdateVersion))
}

// GetUpdateVersion 获取热更新版本
// @Summary 获取热更新版本
// @Description 获取热更新版本
// @Tags version
// @Success 200 {object} web.Response{data=GetUpdateVersionResponse} "成功"
// @Router /version/GetUpdateVersion [get]
func (h *ContentHandler) GetUpdateVersion(ctx *gin.Context) (web.Response, error) {
	resp, err := h.contentClient.GetUpdateVersion(ctx, &contentv1.GetUpdateVersionRequest{})
	if err != nil {
		return web.Response{}, errs.GET_UPDATE_VERSION_ERROR(err)
	}

	var data GetUpdateVersionResponse
	err = copier.Copy(&data.Version, resp.Version)
	if err != nil {
		return web.Response{}, errs.TYPE_CHANGE_ERROR(err)
	}

	return web.Response{
		Msg:  "Success",
		Data: data,
	}, nil
}
