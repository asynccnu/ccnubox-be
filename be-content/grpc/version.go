package grpc

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-content/pkg/errorx"
	contentv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/content/v1"
)

var (
	GET_UPDATE_VERSION_ERROR = errorx.FormatGRPCErrorFunc(contentv1.ErrorGetUpdateServiceError("获取热更新版本失败"))
)

func (c *ContentServiceServer) GetUpdateVersion(ctx context.Context, request *contentv1.GetUpdateVersionRequest) (*contentv1.GetUpdateVersionResponse, error) {
	version := c.svcVersion.GetVersion(ctx)
	return &contentv1.GetUpdateVersionResponse{
		Version: version,
	}, nil
}
