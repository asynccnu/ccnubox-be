package grpc

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-ccnu/service"
	ccnuv1 "github.com/asynccnu/ccnubox-be/common/be-api/gen/proto/ccnu/v1"
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

func (s *CCNUServiceServer) GetXKCookie(ctx context.Context, request *ccnuv1.GetXKCookieRequest) (*ccnuv1.GetXKCookieResponse, error) {
	var (
		cookie string
		err    error
	)
	if request.Type == "" {
		cookie, err = s.ccnu.GetXKCookie(ctx, request.GetStudentId(), request.GetPassword())
	} else {
		cookie, err = s.ccnu.GetXKCookie(ctx, request.GetStudentId(), request.GetPassword(), request.Type) // 这里传入了type, 判断用哪一个系统的cookie
	}
	return &ccnuv1.GetXKCookieResponse{Cookie: cookie}, err
}

func (s *CCNUServiceServer) LoginCCNU(ctx context.Context, request *ccnuv1.LoginCCNURequest) (*ccnuv1.LoginCCNUResponse, error) {
	success, err := s.ccnu.LoginCCNU(ctx, request.GetStudentId(), request.GetPassword())
	return &ccnuv1.LoginCCNUResponse{Success: success}, err
}

func (s *CCNUServiceServer) GetLibraryCookie(ctx context.Context, request *ccnuv1.GetLibraryCookieRequest) (*ccnuv1.GetLibraryCookieResponse, error) {
	cookie, err := s.ccnu.GetLibraryCookie(ctx, request.GetStudentId(), request.GetPassword())
	return &ccnuv1.GetLibraryCookieResponse{Cookie: cookie}, err
}
