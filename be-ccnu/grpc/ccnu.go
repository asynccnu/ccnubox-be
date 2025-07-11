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

func (s *CCNUServiceServer) GetXKCookie(ctx context.Context, request *ccnuv1.GetXKCookieRequest) (*ccnuv1.GetXKCookieResponse, error) {
	cookie, err := s.ccnu.GetXKCookie(ctx, request.GetStudentId(), request.GetPassword())
	return &ccnuv1.GetXKCookieResponse{Cookie: cookie}, err
}

func (s *CCNUServiceServer) GetCCNUCookie(ctx context.Context, request *ccnuv1.GetCCNUCookieRequest) (*ccnuv1.GetCCNUCookieResponse, error) {
	cookie, err := s.ccnu.GetCCNUCookie(ctx, request.GetStudentId(), request.GetPassword())
	return &ccnuv1.GetCCNUCookieResponse{Cookie: cookie}, err
}
