package client

import (
	"context"
	ccnuv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/ccnu/v1"
	"google.golang.org/grpc"
)

type RetryCCNUClient struct {
	ccnuv1.CCNUServiceClient
	retryCnt int
}

func NewRetryCCNUClient(CCNUServiceClient ccnuv1.CCNUServiceClient, retryCnt int) *RetryCCNUClient {
	return &RetryCCNUClient{CCNUServiceClient: CCNUServiceClient, retryCnt: retryCnt}
}

// 兜底的登录机制
func (r *RetryCCNUClient) Login(ctx context.Context, in *ccnuv1.LoginRequest, opts ...grpc.CallOption) (*ccnuv1.LoginResponse, error) {
	var (
		res *ccnuv1.LoginResponse
		err error
	)
	for i := 0; i < r.retryCnt; i++ {
		res, err = r.CCNUServiceClient.Login(ctx, in, opts...)
		if err == nil || ccnuv1.IsInvalidSidOrPwd(err) {
			return res, err
		}
	}
	return nil, err
}
