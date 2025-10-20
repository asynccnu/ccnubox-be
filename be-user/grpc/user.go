package grpc

import (
	"context"
	userv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/user/v1"
	"github.com/asynccnu/ccnubox-be/be-user/service"
	"google.golang.org/grpc"
)

type UserServiceServer struct {
	userv1.UnimplementedUserServiceServer
	svc service.UserService
}

func NewUserServiceServer(svc service.UserService) *UserServiceServer {
	return &UserServiceServer{svc: svc}
}

func (s *UserServiceServer) Register(server grpc.ServiceRegistrar) {
	userv1.RegisterUserServiceServer(server, s)
}

func (s *UserServiceServer) SaveUser(ctx context.Context,
	request *userv1.SaveUserReq) (*userv1.SaveUserResp, error) {
	err := s.svc.Save(ctx, request.GetStudentId(), request.GetPassword())
	return &userv1.SaveUserResp{}, err
}

func (s *UserServiceServer) GetCookie(ctx context.Context, request *userv1.GetCookieRequest) (*userv1.GetCookieResponse, error) {
	u, err := s.svc.GetCookie(ctx, request.GetStudentId())
	return &userv1.GetCookieResponse{Cookie: u}, err
}

func (s *UserServiceServer) CheckUser(ctx context.Context, req *userv1.CheckUserReq) (*userv1.CheckUserResp, error) {
	success, err := s.svc.Check(ctx, req.StudentId, req.Password)

	return &userv1.CheckUserResp{
		Success: success,
	}, err
}
