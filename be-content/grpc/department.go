package grpc

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-content/domain"
	contentv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/content/v1"
)

func (c *ContentServiceServer) GetDepartments(ctx context.Context, request *contentv1.GetDepartmentsRequest) (*contentv1.GetDepartmentsResponse, error) {
	depts, err := c.svcDepartment.GetList(ctx)
	if err != nil {
		// 统一包装为带业务错误码的 RPC 错误
		return nil, err
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
		return nil, err
	}
	return &contentv1.SaveDepartmentResponse{}, nil
}

func (c *ContentServiceServer) DelDepartment(ctx context.Context, request *contentv1.DelDepartmentRequest) (*contentv1.DelDepartmentResponse, error) {
	err := c.svcDepartment.Del(ctx, request.GetId())
	if err != nil {
		return nil, err
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
