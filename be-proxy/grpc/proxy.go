package grpc

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-proxy/service"
	proxyv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/proxy/v1"
	"google.golang.org/grpc"
)

type ProxyServiceServer struct {
	proxyv1.UnimplementedProxyServer
	svc service.ProxyService
}

func NewProxyServiceServer(svc service.ProxyService) *ProxyServiceServer {
	return &ProxyServiceServer{svc: svc}
}

func (s *ProxyServiceServer) Register(server grpc.ServiceRegistrar) {
	proxyv1.RegisterProxyServer(server, s)
}

func (s *ProxyServiceServer) GetProxyAddr(ctx context.Context,
	_ *proxyv1.GetProxyAddrRequest) (*proxyv1.GetProxyAddrResponse, error) {
	res, err := s.svc.GetProxyAddr(ctx)
	if err != nil {
		// 这里就算报错也不能返回空响应, 防止报错, 下游返回的是一个“”, 也可以用
		return &proxyv1.GetProxyAddrResponse{
			Addr: res,
		}, err
	}

	return &proxyv1.GetProxyAddrResponse{Addr: res}, nil
}
