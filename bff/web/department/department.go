package department

import (
	"fmt"
	departmentv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/department/v1"
	"github.com/asynccnu/ccnubox-be/bff/errs"
	"github.com/asynccnu/ccnubox-be/bff/pkg/ginx"
	"github.com/asynccnu/ccnubox-be/bff/web"
	"github.com/asynccnu/ccnubox-be/bff/web/ijwt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
)

type DepartmentHandler struct {
	departmentClient departmentv1.DepartmentServiceClient
	Administrators   map[string]struct{}
}

func NewDepartmentHandler(departmentClient departmentv1.DepartmentServiceClient,
	administrators map[string]struct{}) *DepartmentHandler {
	return &DepartmentHandler{departmentClient: departmentClient, Administrators: administrators}
}

func (h *DepartmentHandler) RegisterRoutes(s *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	sg := s.Group("/department")
	sg.GET("/getDepartments", ginx.Wrap(h.GetDepartments))
	sg.POST("/saveDepartment", authMiddleware, ginx.WrapClaimsAndReq(h.SaveDepartment))
	sg.DELETE("/delDepartment", authMiddleware, ginx.WrapClaimsAndReq(h.DelDepartment))
}

// GetDepartments 获取部门列表
// @Summary 获取部门列表
// @Description 获取部门列表
// @Tags department
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} web.Response{data=GetDepartmentsResponse} "成功"
// @Router /department/getDepartments [get]
func (h *DepartmentHandler) GetDepartments(ctx *gin.Context) (web.Response, error) {
	departments, err := h.departmentClient.GetDepartments(ctx, &departmentv1.GetDepartmentsRequest{})
	if err != nil {
		return web.Response{}, errs.GET_DEPARTMENT_ERROR(err)
	}

	//类型转换
	var resp GetDepartmentsResponse
	err = copier.Copy(&resp.Departments, &departments.Departments)
	if err != nil {
		return web.Response{}, errs.TYPE_CHANGE_ERROR(err)
	}

	return web.Response{
		Msg:  "Success",
		Data: resp,
	}, nil
}

// SaveDepartment 保存部门信息
// @Summary 保存部门信息
// @Description 保存部门信息
// @Tags department
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param request body SaveDepartmentRequest true "保存部门信息请求参数"
// @Success 200 {object} web.Response "成功"
// @Router /department/saveDepartment [post]
func (h *DepartmentHandler) SaveDepartment(ctx *gin.Context, req SaveDepartmentRequest, uc ijwt.UserClaims) (web.Response, error) {
	if !h.isAdmin(uc.StudentId) {
		return web.Response{}, errs.ROLE_ERROR(fmt.Errorf("没有访问权限: %s", uc.StudentId))
	}
	_, err := h.departmentClient.SaveDepartment(ctx, &departmentv1.SaveDepartmentRequest{
		Department: &departmentv1.Department{
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
// @Param Authorization header string true "Bearer Token"
// @Param request body DelDepartmentRequest true "删除部门信息请求参数"
// @Success 200 {object} web.Response "成功"
// @Router /department/delDepartment [delete]
func (h *DepartmentHandler) DelDepartment(ctx *gin.Context, req DelDepartmentRequest, uc ijwt.UserClaims) (web.Response, error) {
	if !h.isAdmin(uc.StudentId) {
		return web.Response{}, errs.ROLE_ERROR(fmt.Errorf("没有访问权限: %s", uc.StudentId))
	}

	_, err := h.departmentClient.DelDepartment(ctx, &departmentv1.DelDepartmentRequest{Id: req.Id})
	if err != nil {
		return web.Response{}, errs.DEL_DEPARTMENT_ERROR(err)
	}
	return web.Response{
		Msg: "Success",
	}, nil
}

func (h *DepartmentHandler) isAdmin(studentId string) bool {
	_, exists := h.Administrators[studentId]
	return exists
}
