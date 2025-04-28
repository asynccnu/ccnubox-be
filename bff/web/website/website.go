package website

import (
	"fmt"
	websitev1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/website/v1"
	"github.com/asynccnu/ccnubox-be/bff/errs"
	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/web"
	"github.com/asynccnu/ccnubox-be/bff/web/department"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
)

type WebsiteHandler struct {
	websiteClient  websitev1.WebsiteServiceClient
	Administrators map[string]struct{}
}

func NewWebsiteHandler(websiteClient websitev1.WebsiteServiceClient,
	administrators map[string]struct{}) *WebsiteHandler {
	return &WebsiteHandler{websiteClient: websiteClient, Administrators: administrators}
}

func (h *WebsiteHandler) RegisterRoutes(s *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	sg := s.Group("/website")
	sg.GET("/getWebsites", ginx.Wrap(h.GetWebsites))
	sg.POST("/saveWebsite", authMiddleware, ginx.WrapClaimsAndReq(h.SaveWebsite))
	sg.DELETE("/delWebsite", authMiddleware, ginx.WrapClaimsAndReq(h.DelWebsite))
}

// GetWebsites 获取网站列表
// @Summary 获取网站列表
// @Description 获取所有网站的列表
// @Tags website
// @Success 200 {object} web.Response{data=GetWebsitesResponse} "成功"
// @Router /website/getWebsites [get]
func (h *WebsiteHandler) GetWebsites(ctx *gin.Context) (web.Response, error) {
	websites, err := h.websiteClient.GetWebsites(ctx, &websitev1.GetWebsitesRequest{})
	if err != nil {
		return web.Response{}, errs.GET_WEBSITES_ERROR(err)
	}
	//类型转换
	var resp GetWebsitesResponse
	err = copier.Copy(&resp.Websites, &websites.Websites)
	if err != nil {
		return web.Response{}, errs.TYPE_CHANGE_ERROR(err)
	}
	return web.Response{
		Msg:  "Success",
		Data: resp,
	}, nil
}

// SaveWebsite 保存网站信息
// @Summary 保存网站信息
// @Description 保存网站信息,id是可选字段,如果有就是替换原来的列表里的,如果没有就是存储新的值
// @Tags website
// @Accept json
// @Produce json
// @Param request body SaveWebsiteRequest true "保存网站信息请求参数"
// @Success 200 {object} web.Response "成功"
// @Router /website/saveWebsite [post]
func (h *WebsiteHandler) SaveWebsite(ctx *gin.Context, req SaveWebsiteRequest, uc ijwt.UserClaims) (web.Response, error) {
	if !h.isAdmin(uc.StudentId) {
		return web.Response{}, errs.ROLE_ERROR(fmt.Errorf("没有访问权限: %s", uc.StudentId))
	}
	_, err := h.websiteClient.SaveWebsite(ctx, &websitev1.SaveWebsiteRequest{
		Website: &websitev1.Website{
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
// @Success 200 {object} web.Response "成功"
// @Router /website/delWebsite [delete]
func (h *WebsiteHandler) DelWebsite(ctx *gin.Context, req department.DelDepartmentRequest, uc ijwt.UserClaims) (web.Response, error) {
	if !h.isAdmin(uc.StudentId) {
		return web.Response{}, errs.ROLE_ERROR(fmt.Errorf("没有访问权限: %s", uc.StudentId))
	}

	_, err := h.websiteClient.DelWebsite(ctx, &websitev1.DelWebsiteRequest{Id: req.Id})
	if err != nil {
		return web.Response{}, errs.DEL_WEBSITE_ERROR(err)
	}
	return web.Response{
		Msg: "Success",
	}, nil
}

func (h *WebsiteHandler) isAdmin(studentId string) bool {
	_, exists := h.Administrators[studentId]
	return exists
}
