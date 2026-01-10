package grpc

import (
	"context"
	"github.com/asynccnu/ccnubox-be/be-content/domain"
	"github.com/asynccnu/ccnubox-be/be-content/pkg/errorx"
	contentv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/content/v1"
)

// 定义 Department 相关的 RPC 错误
var (
	GET_DEPARTMENTS_ERROR = errorx.FormatGRPCErrorFunc(contentv1.ErrorGetDepartmentError("获取部门列表失败"))

	SAVE_DEPARTMENT_ERROR = errorx.FormatGRPCErrorFunc(contentv1.ErrorSaveDepartmentError("保存部门信息失败"))

	DEL_DEPARTMENT_ERROR = errorx.FormatGRPCErrorFunc(contentv1.ErrorDelDepartmentError("删除部门失败"))
)

func (c *ContentServiceServer) GetDepartments(ctx context.Context, request *contentv1.GetDepartmentsRequest) (*contentv1.GetDepartmentsResponse, error) {
	depts, err := c.svcDepartment.GetList(ctx)
	if err != nil {
		// 统一包装为带业务错误码的 RPC 错误
		return nil, GET_DEPARTMENTS_ERROR(err)
	}
	return &contentv1.GetDepartmentsResponse{
		Departments: deptDomains2GRPC(depts),
	}, nil
}

func (c *ContentServiceServer) SaveDepartment(ctx context.Context, request *contentv1.SaveDepartmentRequest) (*contentv1.SaveDepartmentResponse, error) {
	// 转换领域模型并调用 service
	err := c.svcDepartment.Save(ctx, &domain.Department{
		ID:    uint(request.Department.GetId()),
		Name:  request.Department.GetName(),
		Phone: request.Department.GetPhone(),
		Place: request.Department.GetPlace(),
		Time:  request.Department.GetTime(),
	})
	if err != nil {
		return nil, SAVE_DEPARTMENT_ERROR(err)
	}
	return &contentv1.SaveDepartmentResponse{}, nil
}

func (c *ContentServiceServer) DelDepartment(ctx context.Context, request *contentv1.DelDepartmentRequest) (*contentv1.DelDepartmentResponse, error) {
	err := c.svcDepartment.Del(ctx, request.GetId())
	if err != nil {
		return nil, DEL_DEPARTMENT_ERROR(err)
	}
	return &contentv1.DelDepartmentResponse{}, nil
}

func deptDomains2GRPC(depts []domain.Department) []*contentv1.Department {
	res := make([]*contentv1.Department, 0, len(depts))
	for _, d := range depts {
		res = append(res, &contentv1.Department{
			Id:    int64(d.ID),
			Name:  d.Name,
			Phone: d.Phone,
			Place: d.Place,
			Time:  d.Time,
		})
	}
	return res
}
