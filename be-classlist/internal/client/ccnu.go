package client

import (
	"context"
	"fmt"

	"github.com/asynccnu/ccnubox-be/be-classlist/internal/conf"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/errcode"
	user "github.com/asynccnu/ccnubox-be/common/api/gen/proto/user/v1"
	b_conf "github.com/asynccnu/ccnubox-be/common/bizpkg/conf"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
)

type UserSvc struct {
	usc user.UserServiceClient
}

func NewUserSvc(r *etcd.Registry, cf *conf.Registry, env *b_conf.Env) (*UserSvc, error) {
	conn, err := InitClient(r, cf.Usersvc, env)
	if err != nil {
		return nil, err
	}
	usc := user.NewUserServiceClient(conn)
	return &UserSvc{usc: usc}, nil
}

func (c *UserSvc) GetCookie(ctx context.Context, stuID string) (string, error) {

	req := &user.GetCookieRequest{
		StudentId: stuID,
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
