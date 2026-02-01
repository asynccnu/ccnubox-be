package content

import (
	"fmt"

	"github.com/asynccnu/ccnubox-be/bff/errs"
	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/web"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	contentv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/content/v1"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
)

func (h *ContentHandler) RegisterUpdateVersionRoute(group *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	sg := group.Group("/version")
	sg.GET("/getVersion", ginx.Wrap(h.GetUpdateVersion))
	sg.POST("/saveVersion", authMiddleware, ginx.WrapClaimsAndReq(h.SaveUpdateVersion))
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

// SaveUpdateVersion 更新热更新版本
// @Summary 更新热更新版本
// @Description 更新热更新版本
// @Tags version
// @Accept json
// @Produce json
// @Param request body SaveVersionRequest true "保存版本号请求参数"
// @Success 200 {object} web.Response "成功"
// @Router /version/saveVersion [post]
func (h *ContentHandler) SaveUpdateVersion(ctx *gin.Context, req SaveVersionRequest, uc ijwt.UserClaims) (web.Response, error) {
	if !h.isAdmin(uc.StudentId) {
		return web.Response{}, errs.ROLE_ERROR(fmt.Errorf("没有访问权限: %s", uc.StudentId))
	}
	_, err := h.contentClient.SaveUpdateVersion(ctx, &contentv1.SaveUpdateVersionRequest{
		Version: req.Version,
	})
	if err != nil {
		return web.Response{}, errs.SAVE_UPDATE_VERSION_ERROR(err)
	}
	return web.Response{
		Msg: "Success",
	}, nil
}
