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

// RegisterWebsiteRoute 注册 Website 相关路由
func (h *ContentHandler) RegisterWebsiteRoute(group *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	sg := group.Group("/website")
	sg.GET("/getWebsites", ginx.Wrap(h.GetWebsites))
	sg.POST("/saveWebsite", authMiddleware, ginx.WrapClaimsAndReq(h.SaveWebsite))
	sg.POST("/delWebsite", authMiddleware, ginx.WrapClaimsAndReq(h.DelWebsite))
}

// GetWebsites 获取网站列表
// @Summary 获取网站列表
// @Description 获取所有网站的列表
// @Tags website
// @Success 200 {object} web.Response{data=GetWebsitesResponse} "成功"
// @Router /website/getWebsites [get]
func (h *ContentHandler) GetWebsites(ctx *gin.Context) (web.Response, error) {
	// 统一调用聚合后的 contentClient
	resp, err := h.contentClient.GetWebsites(ctx, &contentv1.GetWebsitesRequest{})
	if err != nil {
		return web.Response{}, errs.GET_WEBSITES_ERROR(err)
	}

	// 类型转换
	var data GetWebsitesResponse
	if err := copier.Copy(&data.Websites, &resp.Websites); err != nil {
		return web.Response{}, errs.TYPE_CHANGE_ERROR(err)
	}

	return web.Response{
		Msg:  "Success",
		Data: data,
	}, nil
}

// SaveWebsite 保存网站信息
// @Summary 保存网站信息
// @Description 保存网站信息,id是可选字段,如果有就是替换原来的列表里的,如果没有就是存储新的值
// @Tags website
// @Accept json
// @Produce json
// @Param request body SaveWebsiteRequest true "保存网站信息请求参数"
// @Success 200 {object} web.Response{data=GetWebsitesResponse} "成功"
// @Router /website/saveWebsite [post]
func (h *ContentHandler) SaveWebsite(ctx *gin.Context, req SaveWebsiteRequest, uc ijwt.UserClaims) (web.Response, error) {
	if !h.isAdmin(uc.StudentId) {
		return web.Response{}, errs.ROLE_ERROR(fmt.Errorf("没有访问权限: %s", uc.StudentId))
	}

	// 后端 SaveWebsite 会返回全量列表
	_, err := h.contentClient.SaveWebsite(ctx, &contentv1.SaveWebsiteRequest{
		Website: &contentv1.Website{
			Id:          req.Id,
			Link:        req.Link,
			Name:        req.Name,
			Description: req.Description,
			Image:       req.Image,
		},
	})
	if err != nil {
		return web.Response{}, errs.SAVE_WEBSITE_ERROR(err)
	}

	return web.Response{
		Msg: "Success",
	}, nil
}

// DelWebsite 删除网站信息
// @Summary 删除网站信息
// @Description 删除网站信息
// @Tags website
// @Accept json
// @Produce json
// @Param request body DelWebsiteRequest true "删除网站信息请求参数"
// @Success 200 {object} web.Response{data=GetWebsitesResponse} "成功"
// @Router /website/delWebsite [post]
func (h *ContentHandler) DelWebsite(ctx *gin.Context, req DelWebsiteRequest, uc ijwt.UserClaims) (web.Response, error) {
	if !h.isAdmin(uc.StudentId) {
		return web.Response{}, errs.ROLE_ERROR(fmt.Errorf("没有访问权限: %s", uc.StudentId))
	}

	// 后端 DelWebsite 会返回全量列表
	_, err := h.contentClient.DelWebsite(ctx, &contentv1.DelWebsiteRequest{Id: req.Id})
	if err != nil {
		return web.Response{}, errs.DEL_WEBSITE_ERROR(err)
	}
	return web.Response{
		Msg: "Success",
	}, nil
}
