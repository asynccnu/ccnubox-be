package grpc

import (
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/service"
	classlistv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/classlist/v1"
	"google.golang.org/grpc"
)

type ClasslistServiceServer struct {
	classlistv1.UnimplementedClasserServer
	svc *service.ClassListService
}

func NewCalendarServiceServer(svc *service.ClassListService) *ClasslistServiceServer {
	return &ClasslistServiceServer{
		svc: svc,
	}
}

// 注册为grpc服务
func (c *ClasslistServiceServer) Register(server grpc.ServiceRegistrar) {
	classlistv1.RegisterClasserServer(server, c)
}
