package grpc

import (
	"context"
	departmentv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/department/v1"
	"github.com/asynccnu/ccnubox-be/be-department/domain"
	"github.com/asynccnu/ccnubox-be/be-department/service"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

type DepartmentServiceServer struct {
	departmentv1.UnimplementedDepartmentServiceServer
	svc service.DepartmentService
}

func NewDepartmentServiceServer(svc service.DepartmentService) *DepartmentServiceServer {
	return &DepartmentServiceServer{svc: svc}
}

func (d *DepartmentServiceServer) GetDepartments(ctx context.Context, request *departmentv1.GetDepartmentsRequest) (*departmentv1.GetDepartmentsResponse, error) {
	departments, err := d.svc.GetDepartments(ctx)
	if err != nil {
		return nil, err
	}
	var resp departmentv1.GetDepartmentsResponse
	for _, department := range departments {
		resp.Departments = append(resp.Departments, &departmentv1.Department{
			Name:  department.Name,
			Phone: department.Phone,
			Place: department.Place,
			Time:  department.Time,
			Id:    int64(department.ID),
		})
	}
	return &resp, nil
}

func (d *DepartmentServiceServer) SaveDepartment(ctx context.Context, request *departmentv1.SaveDepartmentRequest) (*departmentv1.SaveDepartmentResponse, error) {
	err := d.svc.SaveDepartment(ctx, &domain.Department{
		Name:  request.Department.Name,
		Phone: request.Department.Phone,
		Place: request.Department.Place,
		Time:  request.Department.Time,
	})
	if err != nil {
		return nil, err
	}
	return &departmentv1.SaveDepartmentResponse{}, nil
}

func (d *DepartmentServiceServer) DelDepartment(ctx context.Context, request *departmentv1.DelDepartmentRequest) (*departmentv1.DelDepartmentResponse, error) {
	err := d.svc.DelDepartment(ctx, uint(request.GetId()))
	if err != nil {
		return nil, err
	}
	return &departmentv1.DelDepartmentResponse{}, nil
}

// 注册为grpc服务
func (d *DepartmentServiceServer) Register(server *grpc.Server) {
	departmentv1.RegisterDepartmentServiceServer(server, d)
}
