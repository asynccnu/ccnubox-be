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

// RegisterInfoSumRoute 注册 InfoSum 相关路由
func (h *ContentHandler) RegisterInfoSumRoute(group *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	sg := group.Group("/InfoSum")
	sg.GET("/getInfoSums", ginx.Wrap(h.GetInfoSums))
	sg.POST("/saveInfoSum", authMiddleware, ginx.WrapClaimsAndReq(h.SaveInfoSum))
	sg.POST("/delInfoSum", authMiddleware, ginx.WrapClaimsAndReq(h.DelInfoSum))
}

// GetInfoSums 获取信息整合列表
// @Summary 获取信息整合列表
// @Description 获取所有信息整合的列表
// @Tags InfoSum
// @Success 200 {object} web.Response{data=GetInfoSumsResponse} "成功"
// @Router /InfoSum/getInfoSums [get]
func (h *ContentHandler) GetInfoSums(ctx *gin.Context) (web.Response, error) {
	resp, err := h.contentClient.GetInfoSums(ctx, &contentv1.GetInfoSumsRequest{})
	if err != nil {
		return web.Response{}, errs.GET_INFOSUM_ERROR(err)
	}

	var data GetInfoSumsResponse
	if err := copier.Copy(&data.InfoSums, &resp.InfoSums); err != nil {
		return web.Response{}, errs.TYPE_CHANGE_ERROR(err)
	}

	return web.Response{
		Msg:  "Success",
		Data: data,
	}, nil
}

// SaveInfoSum 保存信息整合信息
// @Summary 保存信息整合信息
// @Description 保存信息整合信息,id是可选字段,如果有就是替换原来的列表里的,如果没有就是存储新的值
// @Tags InfoSum
// @Accept json
// @Produce json
// @Param request body SaveInfoSumRequest true "保存信息整合信息请求参数"
// @Success 200 {object} web.Response{data=GetInfoSumsResponse} "成功"
// @Router /InfoSum/saveInfoSum [post]
func (h *ContentHandler) SaveInfoSum(ctx *gin.Context, req SaveInfoSumRequest, uc ijwt.UserClaims) (web.Response, error) {
	if !h.isAdmin(uc.StudentId) {
		return web.Response{}, errs.ROLE_ERROR(fmt.Errorf("没有访问权限: %s", uc.StudentId))
	}

	// 调用后端，后端会返回更新后的全量列表
	_, err := h.contentClient.SaveInfoSum(ctx, &contentv1.SaveInfoSumRequest{
		InfoSum: &contentv1.InfoSum{
			Id:          req.Id,
			Link:        req.Link,
			Name:        req.Name,
			Description: req.Description,
			Image:       req.Image,
		},
	})
	if err != nil {
		return web.Response{}, errs.Save_INFOSUM_ERROR(err)
	}

	return web.Response{
		Msg: "Success",
	}, nil
}

// DelInfoSum 删除信息整合信息
// @Summary 删除信息整合信息
// @Description 删除信息整合信息
// @Tags InfoSum
// @Accept json
// @Produce json
// @Param request body DelInfoSumRequest true "删除信息整合信息请求参数"
// @Success 200 {object} web.Response{data=GetInfoSumsResponse} "成功"
// @Router /InfoSum/delInfoSum [post]
func (h *ContentHandler) DelInfoSum(ctx *gin.Context, req DelInfoSumRequest, uc ijwt.UserClaims) (web.Response, error) {
	if !h.isAdmin(uc.StudentId) {
		return web.Response{}, errs.ROLE_ERROR(fmt.Errorf("没有访问权限: %s", uc.StudentId))
	}

	_, err := h.contentClient.DelInfoSum(ctx, &contentv1.DelInfoSumRequest{Id: req.Id})
	if err != nil {
		return web.Response{}, errs.Del_INFOSUM_ERROR(err)
	}

	return web.Response{
		Msg: "Success",
	}, nil
}
