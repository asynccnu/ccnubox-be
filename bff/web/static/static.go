package static

import (
	"errors"
	"fmt"
	staticv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/static/v1"
	"github.com/asynccnu/ccnubox-be/bff/errs"
	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/pkg/htmlx"
	"github.com/asynccnu/ccnubox-be/bff/web"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
)

type StaticHandler struct {
	staticClient           staticv1.StaticServiceClient
	fileToHTMLConverterMap map[string]htmlx.FileToHTMLConverter
	Administrators         map[string]struct{}
}

func NewStaticHandler(
	staticClient staticv1.StaticServiceClient,
	fileToHTMLConverterMap map[string]htmlx.FileToHTMLConverter,
	administrators map[string]struct{},
) *StaticHandler {
	return &StaticHandler{staticClient: staticClient, fileToHTMLConverterMap: fileToHTMLConverterMap, Administrators: administrators}
}

func (h *StaticHandler) RegisterRoutes(s *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	sg := s.Group("/statics")
	sg.GET("", ginx.WrapReq(h.GetStaticByName))
	sg.GET("/match/labels", ginx.Wrap(h.GetStaticByLabels))
	sg.POST("/save", authMiddleware, ginx.WrapClaimsAndReq(h.SaveStatic))
}

// GetStaticByName
// @Summary 获取静态资源[精确名称]
// @Description 【弃用】根据静态资源名称获取静态资源的内容。
// @Tags statics[Deprecation]
// @Accept json
// @Produce json
// @Param static_name query string true "静态资源名称"
// @Success 200 {object} web.Response{data=StaticVo} "成功"
// @Router /statics [get]
func (h *StaticHandler) GetStaticByName(ctx *gin.Context, req GetStaticByNameReq) (web.Response, error) {
	if req.StaticName == "" {
		return web.Response{}, errs.INVALID_PARAM_VALUE_ERROR(errors.New("静态名称不合法"))
	}
	res, err := h.staticClient.GetStaticByName(ctx, &staticv1.GetStaticByNameRequest{Name: req.StaticName})

	if err != nil {
		return web.Response{
			Code: errs.INTERNAL_SERVER_ERROR_CODE,
			Msg:  "系统异常",
		}, err
	}
	var resp StaticVo
	err = copier.Copy(&resp, &res)
	if err != nil {
		return web.Response{}, errs.TYPE_CHANGE_ERROR(err)
	}
	return web.Response{
		Msg:  "Success",
		Data: resp,
	}, nil
}

// SaveStatic
// @Summary 保存静态内容
// @Description 【弃用】保存静态内容
// @Tags statics[Deprecation]
// @Accept json
// @Produce json
// @Param request body SaveStaticReq true "保存静态内容请求参数"
// @Success 200 {object} web.Response "成功"
// @Router /statics/save [post]
func (h *StaticHandler) SaveStatic(ctx *gin.Context, req SaveStaticReq, uc ijwt.UserClaims) (web.Response, error) {
	// 管理员身份验证
	if !h.isAdmin(uc.StudentId) {
		return web.Response{}, errs.SAVE_STATIC_ERROR(fmt.Errorf("没有访问权限: %s", uc.StudentId))
	}
	if req.Name == "" {
		return web.Response{}, errs.INVALID_PARAM_VALUE_ERROR(errors.New("静态名称不合法"))
	}
	_, err := h.staticClient.SaveStatic(ctx, &staticv1.SaveStaticRequest{
		Static: &staticv1.Static{
			Name:    req.Name,
			Content: req.Content,
			Labels:  req.Labels,
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

func (h *StaticHandler) isAdmin(studentId string) bool {
	_, exists := h.Administrators[studentId]
	return exists
}

// GetStaticByLabels
// @Summary 获取静态资源[标签匹配]
// @Description【弃用】根据静labels匹配合适的静态资源
// @Tags statics[Deprecation]
// @Accept multipart/form-data
// @Produce json
// @Param labels[type] query string true "标签：标明匹配哪一类的资源"
// @Success 200 {object} web.Response{data=GetStaticByLabelsResp} "成功"
// @Router /statics/match/labels [get]
func (h *StaticHandler) GetStaticByLabels(ctx *gin.Context) (web.Response, error) {
	labels := ctx.QueryMap("labels")
	if len(labels) == 0 {
		return web.Response{}, errs.INVALID_PARAM_VALUE_ERROR(errors.New("空map"))
	}
	res, err := h.staticClient.GetStaticsByLabels(ctx, &staticv1.GetStaticsByLabelsRequest{
		Labels: labels,
	})
	if err != nil {
		return web.Response{
			Code: errs.INTERNAL_SERVER_ERROR_CODE,
			Msg:  "系统异常",
		}, errs.GET_STATIC_BY_LABELS_ERROR(err)
	}
	var resp GetStaticByLabelsResp
	err = copier.Copy(&resp.Statics, &res)
	if err != nil {
		return web.Response{}, errs.TYPE_CHANGE_ERROR(err)
	}
	return web.Response{
		Msg:  "Success",
		Data: resp,
	}, nil
}
