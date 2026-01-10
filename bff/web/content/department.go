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

// RegisterDepartmentRoute 注册 Department 相关路由
func (h *ContentHandler) RegisterDepartmentRoute(group *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	sg := group.Group("/department")
	sg.GET("/getDepartments", ginx.Wrap(h.GetDepartments))
	sg.POST("/saveDepartment", authMiddleware, ginx.WrapClaimsAndReq(h.SaveDepartment))
	sg.POST("/delDepartment", authMiddleware, ginx.WrapClaimsAndReq(h.DelDepartment))
}

// GetDepartments 获取部门列表
// @Summary 获取部门列表
// @Description 获取部门列表
// @Tags department
// @Success 200 {object} web.Response{data=GetDepartmentsResponse} "成功"
// @Router /department/getDepartments [get]
func (h *ContentHandler) GetDepartments(ctx *gin.Context) (web.Response, error) {
	// 调用聚合后的 contentClient
	resp, err := h.contentClient.GetDepartments(ctx, &contentv1.GetDepartmentsRequest{})
	if err != nil {
		return web.Response{}, errs.GET_DEPARTMENT_ERROR(err)
	}

	// 类型转换
	var data GetDepartmentsResponse
	err = copier.Copy(&data.Departments, &resp.Departments)
	if err != nil {
		return web.Response{}, errs.TYPE_CHANGE_ERROR(err)
	}

	return web.Response{
		Msg:  "Success",
		Data: data,
	}, nil
}

// SaveDepartment 保存部门信息
// @Summary 保存部门信息
// @Description 保存部门信息
// @Tags department
// @Accept json
// @Produce json
// @Param request body SaveDepartmentRequest true "保存部门信息请求参数"
// @Success 200 {object} web.Response "成功"
// @Router /department/saveDepartment [post]
func (h *ContentHandler) SaveDepartment(ctx *gin.Context, req SaveDepartmentRequest, uc ijwt.UserClaims) (web.Response, error) {
	// 复用 ContentHandler 的 isAdmin
	if !h.isAdmin(uc.StudentId) {
		return web.Response{}, errs.ROLE_ERROR(fmt.Errorf("没有访问权限: %s", uc.StudentId))
	}

	_, err := h.contentClient.SaveDepartment(ctx, &contentv1.SaveDepartmentRequest{
		Department: &contentv1.Department{
			Id:    req.Id,
			Name:  req.Name,
			Phone: req.Phone,
			Place: req.Place,
			Time:  req.Time,
		},
	})
	if err != nil {
		return web.Response{}, errs.SAVE_DEPARTMENT_ERROR(err)
	}

	return web.Response{
		Msg: "Success",
	}, nil
}

// DelDepartment 删除部门信息
// @Summary 删除部门信息
// @Description 删除部门信息
// @Tags department
// @Accept json
// @Produce json
// @Param request body DelDepartmentRequest true "删除部门信息请求参数"
// @Success 200 {object} web.Response "成功"
// @Router /department/delDepartment [post]
func (h *ContentHandler) DelDepartment(ctx *gin.Context, req DelDepartmentRequest, uc ijwt.UserClaims) (web.Response, error) {
	if !h.isAdmin(uc.StudentId) {
		return web.Response{}, errs.ROLE_ERROR(fmt.Errorf("没有访问权限: %s", uc.StudentId))
	}

	_, err := h.contentClient.DelDepartment(ctx, &contentv1.DelDepartmentRequest{
		Id: req.Id,
	})
	if err != nil {
		return web.Response{}, errs.DEL_DEPARTMENT_ERROR(err)
	}

	return web.Response{
		Msg: "Success",
	}, nil
}
