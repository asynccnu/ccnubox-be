package grpc

import (
	"context"

	contentv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/content/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
)

var (
	SAVE_UPDATE_VERSION_ERROR = errorx.FormatErrorFunc(contentv1.ErrorGetUpdateServiceError("保存热更新版本失败"))
)

func (c *ContentServiceServer) GetUpdateVersion(ctx context.Context, request *contentv1.GetUpdateVersionRequest) (*contentv1.GetUpdateVersionResponse, error) {
	version := c.svcVersion.Get(ctx)
	return &contentv1.GetUpdateVersionResponse{
		Version: version,
	}, nil
}

func (c *ContentServiceServer) SaveUpdateVersion(ctx context.Context, request *contentv1.SaveUpdateVersionRequest) (*contentv1.SaveUpdateVersionResponse, error) {
	err := c.svcVersion.Save(ctx, request.Version)
	if err != nil {
		return nil, SAVE_UPDATE_VERSION_ERROR(err)
	}
	return &contentv1.SaveUpdateVersionResponse{}, nil
}
