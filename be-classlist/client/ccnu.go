package client

import (
	"context"
	"fmt"

	"github.com/asynccnu/ccnubox-be/be-classlist/biz"
	userv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/user/v1"
	"github.com/asynccnu/ccnubox-be/common/tool"
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
	if err != nil {
		return "", mapGetCookieError(err)
	}
	if resp == nil || resp.Cookie == "" {
		return "", fmt.Errorf("get cookie from user service: %w", biz.ErrCookieUnavailable)
	}
	return resp.Cookie, nil
}

func mapGetCookieError(err error) error {
	if tool.IsCCNUAccountInitializationRequired(err) {
		return fmt.Errorf("%w: %w", biz.ErrCrawlerAuthentication, tool.ErrCCNUAccountInitializationRequired)
	}
	return fmt.Errorf("get cookie from user service: %w", err)
}
