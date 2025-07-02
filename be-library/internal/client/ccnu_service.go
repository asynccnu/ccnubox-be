package client

import (
	"context"
	"fmt"
	"time"

	user "github.com/asynccnu/ccnubox-be/be-api/gen/proto/user/v1"
	"github.com/asynccnu/ccnubox-be/be-library/internal/biz"
	"github.com/asynccnu/ccnubox-be/be-library/internal/conf"
	"github.com/asynccnu/ccnubox-be/be-library/internal/errcode"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

type CCNUService struct {
	Cs user.UserServiceClient
}

func NewCCNUServiceProxy(cs user.UserServiceClient) biz.CCNUServiceProxy {
	return &CCNUService{Cs: cs}
}

func (c *CCNUService) GetCookie(ctx context.Context, stuID string) (string, error) {
	resp, err := c.Cs.GetCookie(ctx, &user.GetCookieRequest{
		StudentId: stuID,
	})
	if err != nil {
		fmt.Println(err)
		return "", errcode.ErrCCNULogin
	}
	cookie := resp.Cookie
	return cookie, nil
}

func NewClient(r *etcd.Registry, cf *conf.Registry, logger log.Logger) (user.UserServiceClient, error) {
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint(cf.Usersvc), // 需要发现的服务，如果是k8s部署可以直接用服务器本地地址:9001，9001端口是需要调用的服务的端口
		grpc.WithDiscovery(r),
		grpc.WithTimeout(5*time.Second), //由于使用华师的服务,所以设置下超时时间最长为5s
		grpc.WithMiddleware(
			tracing.Client(),
			recovery.Recovery(),
		),
	)
	if err != nil {
		log.NewHelper(logger).WithContext(context.Background()).Errorw("kind", "grpc-client", "reason", "GRPC_CLIENT_INIT_ERROR", "err", err)
		return nil, err
	}
	return user.NewUserServiceClient(conn), nil
}
