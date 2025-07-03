package infoSum

import (
	"fmt"
	InfoSumv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/infoSum/v1"
	"github.com/asynccnu/ccnubox-be/bff/errs"
	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/web"
	"github.com/asynccnu/ccnubox-be/bff/web/department"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
)

type InfoSumHandler struct {
	InfoSumClient  InfoSumv1.InfoSumServiceClient
	Administrators map[string]struct{}
}

func NewInfoSumHandler(InfoSumClient InfoSumv1.InfoSumServiceClient,
	administrators map[string]struct{}) *InfoSumHandler {
	return &InfoSumHandler{InfoSumClient: InfoSumClient, Administrators: administrators}
}

func (h *InfoSumHandler) RegisterRoutes(s *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	sg := s.Group("/InfoSum")
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
func (h *InfoSumHandler) GetInfoSums(ctx *gin.Context) (web.Response, error) {
	InfoSums, err := h.InfoSumClient.GetInfoSums(ctx, &InfoSumv1.GetInfoSumsRequest{})
	if err != nil {
		return web.Response{}, errs.GET_INFOSUM_ERROR(err)
	}
	//类型转换
	var resp GetInfoSumsResponse
	err = copier.Copy(&resp.InfoSums, &InfoSums.InfoSums)
	if err != nil {
		return web.Response{
			Code: errs.INTERNAL_SERVER_ERROR_CODE,
			Msg:  "系统异常",
		}, errs.TYPE_CHANGE_ERROR(err)
	}
	return web.Response{
		Msg:  "Success",
		Data: resp,
	}, nil
}

// SaveInfoSum 保存信息整合信息
// @Summary 保存信息整合信息
// @Description 保存信息整合信息,id是可选字段,如果有就是替换原来的列表里的,如果没有就是存储新的值
// @Tags InfoSum
// @Accept json
// @Produce json
// @Param request body SaveInfoSumRequest true "保存信息整合信息请求参数"
// @Success 200 {object} web.Response "成功"
// @Router /InfoSum/saveInfoSum [post]
func (h *InfoSumHandler) SaveInfoSum(ctx *gin.Context, req SaveInfoSumRequest, uc ijwt.UserClaims) (web.Response, error) {
	if !h.isAdmin(uc.StudentId) {
		return web.Response{}, errs.ROLE_ERROR(fmt.Errorf("没有访问权限: %s", uc.StudentId))
	}

	_, err := h.InfoSumClient.SaveInfoSum(ctx, &InfoSumv1.SaveInfoSumRequest{
		InfoSum: &InfoSumv1.InfoSum{
			Id:          req.Id,
			Link:        req.Link,
			Name:        req.Name,
			Description: req.Description,
			Image:       req.Image,
		},
	})
	if err != nil {
		return web.Response{
			Code: errs.INTERNAL_SERVER_ERROR_CODE,
			Msg:  "系统异常",
		}, err
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
// @Success 200 {object} web.Response "成功"
// @Router /InfoSum/delInfoSum [post]
func (h *InfoSumHandler) DelInfoSum(ctx *gin.Context, req department.DelDepartmentRequest, uc ijwt.UserClaims) (web.Response, error) {
	if !h.isAdmin(uc.StudentId) {
		return web.Response{}, errs.ROLE_ERROR(fmt.Errorf("没有访问权限: %s", uc.StudentId))
	}

	_, err := h.InfoSumClient.DelInfoSum(ctx, &InfoSumv1.DelInfoSumRequest{Id: req.Id})
	if err != nil {
		return web.Response{
			Code: errs.INTERNAL_SERVER_ERROR_CODE,
			Msg:  "系统异常",
		}, errs.Del_INFOSUM_ERROR(err)
	}
	return web.Response{
		Msg: "Success",
	}, nil
}

func (h *InfoSumHandler) isAdmin(studentId string) bool {
	_, exists := h.Administrators[studentId]
	return exists
}
