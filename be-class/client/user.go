package client

import (
	"context"
	"fmt"

	"github.com/asynccnu/ccnubox-be/be-class/biz"
	userv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/user/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/asynccnu/ccnubox-be/common/tool"
)

type UserService struct {
	client userv1.UserServiceClient
	logger logger.Logger
}

func NewUserService(client userv1.UserServiceClient, l logger.Logger) *UserService {
	return &UserService{client: client, logger: l}
}

func (c *UserService) GetCookie(ctx context.Context, studentID string, system tool.XKSystem) (string, error) {
	req := &userv1.GetCookieRequest{StudentId: studentID, Type: string(system)}
	resp, err := c.client.GetCookie(ctx, req)
	if err != nil {
		c.logger.WithContext(ctx).Warn("get cookie failed", logger.String("studentID", studentID), logger.Error(err))
		return "", fmt.Errorf("%s: %w", biz.ErrCCNULogin, err)
	}
	if resp.Cookie == "" {
		return "", fmt.Errorf("cookie is empty")
	}
	return resp.Cookie, nil
}
