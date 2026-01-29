package grpc

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-user/service"
	userv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/user/v1"
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
	var (
		u   string
		err error
	)
	if request.Type == "" {
		u, err = s.svc.GetCookie(ctx, request.GetStudentId())
	} else {
		u, err = s.svc.GetCookie(ctx, request.GetStudentId(), request.Type)

	}
	return &userv1.GetCookieResponse{Cookie: u}, err
}

func (s *UserServiceServer) GetLibraryCookie(ctx context.Context, request *userv1.GetLibraryCookieRequest) (*userv1.GetLibraryCookieResponse, error) {
	cookie, err := s.svc.GetLibraryCookie(ctx, request.GetStudentId())
	return &userv1.GetLibraryCookieResponse{Cookie: cookie}, err
}

func (s *UserServiceServer) CheckUser(ctx context.Context, req *userv1.CheckUserReq) (*userv1.CheckUserResp, error) {
	success, err := s.svc.Check(ctx, req.StudentId, req.Password)

	return &userv1.CheckUserResp{
		Success: success,
	}, err
}
