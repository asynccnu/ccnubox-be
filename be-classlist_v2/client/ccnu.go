package client

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-classlist_v2/biz"
	userv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/user/v1"
)

type CCNUService struct {
	user userv1.UserServiceClient
}

func NewCCNUService(user userv1.UserServiceClient) biz.CCNUService {
	return &CCNUService{user: user}
}

func (c *CCNUService) GetCookie(ctx context.Context, stuID string) (string, error) {
	resp, err := c.user.GetCookie(ctx, &userv1.GetCookieRequest{
		StudentId: stuID,
	})
	return resp.Cookie, err
}
