package grpc

import (
	"context"
	ccnuv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/ccnu/v1"
	"github.com/asynccnu/ccnubox-be/be-ccnu/service"
	"google.golang.org/grpc"
)

type CCNUServiceServer struct {
	ccnuv1.UnimplementedCCNUServiceServer
	ccnu service.CCNUService
}

func NewCCNUServiceServer(ccnu service.CCNUService) *CCNUServiceServer {
	return &CCNUServiceServer{ccnu: ccnu}
}

func (s *CCNUServiceServer) Register(server grpc.ServiceRegistrar) {
	ccnuv1.RegisterCCNUServiceServer(server, s)
}

func (s *CCNUServiceServer) Login(ctx context.Context, request *ccnuv1.LoginRequest) (*ccnuv1.LoginResponse, error) {
	success, err := s.ccnu.Login(ctx, request.GetStudentId(), request.GetPassword())
	return &ccnuv1.LoginResponse{Success: success}, err
}
func (s *CCNUServiceServer) GetCCNUCookie(ctx context.Context, request *ccnuv1.GetCCNUCookieRequest) (*ccnuv1.GetCCNUCookieResponse, error) {
	cookie, err := s.ccnu.GetCCNUCookie(ctx, request.GetStudentId(), request.GetPassword())
	return &ccnuv1.GetCCNUCookieResponse{Cookie: cookie}, err
}
