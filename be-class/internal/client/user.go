package client

import (
	"context"
	"fmt"

	"github.com/asynccnu/ccnubox-be/be-class/internal/conf"
	"github.com/asynccnu/ccnubox-be/be-class/internal/errcode"
	clog "github.com/asynccnu/ccnubox-be/be-class/internal/log"
	user "github.com/asynccnu/ccnubox-be/common/api/gen/proto/user/v1"
	b_conf "github.com/asynccnu/ccnubox-be/common/bizpkg/conf"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
)

type UserSvc struct {
	usc user.UserServiceClient
}

func NewUserSvc(r *etcd.Registry, cf *conf.Registry, env *b_conf.Env) (*UserSvc, error) {
	clog.LogPrinter.Infof("init user grpc client, target service address: %s", cf.Usersvc)
	conn, err := InitClient(r, cf.Usersvc, env)
	if err != nil {
		clog.LogPrinter.Errorw("kind", "grpc-client", "reason", "GRPC_CLIENT_INIT_ERROR", "err", err)
		return nil, err
	}
	usc := user.NewUserServiceClient(conn)
	clog.LogPrinter.Infof("init user grpc client success, address: %s", cf.Usersvc)
	return &UserSvc{usc: usc}, nil
}

func (c *UserSvc) GetCookie(ctx context.Context, stuID string, tpe ...string) (string, error) {

	req := &user.GetCookieRequest{
		StudentId: stuID,
	}
	if len(tpe) != 0 {
		req.Type = tpe[0]
	}
	resp, err := c.usc.GetCookie(ctx, req)
	if err != nil {
		return "", fmt.Errorf("%s:%v", errcode.ErrCCNULogin, err)
	}
	cookie := resp.Cookie
	if len(cookie) == 0 {
		return "", fmt.Errorf("cookie is empty")
	}
	return cookie, nil
}
