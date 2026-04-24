package client

import (
	"context"
	"fmt"

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
	if err != nil || resp == nil {
		return "", fmt.Errorf("No cookie was fetched from user service err=%v", err)
	}
	return resp.Cookie, nil
}
