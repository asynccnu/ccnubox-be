package grpc

import (
	"context"
	"github.com/asynccnu/ccnubox-be/be-content/domain"
	"github.com/asynccnu/ccnubox-be/be-content/pkg/errorx"
	contentv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/content/v1"
)

// 定义 InfoSum 相关的 RPC 错误
var (
	GET_INFOSUMS_ERROR = errorx.FormatGRPCErrorFunc(contentv1.ErrorGetInfosumError("获取信息汇总列表失败"))

	SAVE_INFOSUM_ERROR = errorx.FormatGRPCErrorFunc(contentv1.ErrorSaveInfosumError("保存信息汇总失败"))

	DEL_INFOSUM_ERROR = errorx.FormatGRPCErrorFunc(contentv1.ErrorDelInfosumError("删除信息汇总失败"))
)

func (c *ContentServiceServer) GetInfoSums(ctx context.Context, request *contentv1.GetInfoSumsRequest) (*contentv1.GetInfoSumsResponse, error) {
	infos, err := c.svcInfoSum.GetList(ctx)
	if err != nil {
		return nil, GET_INFOSUMS_ERROR(err)
	}
	return &contentv1.GetInfoSumsResponse{
		InfoSums: infoSumDomains2GRPC(infos),
	}, nil
}

func (c *ContentServiceServer) SaveInfoSum(ctx context.Context, request *contentv1.SaveInfoSumRequest) (*contentv1.SaveInfoSumResponse, error) {
	// 1. 执行保存
	err := c.svcInfoSum.Save(ctx, &domain.InfoSum{
		ID:          uint(request.InfoSum.GetId()),
		Name:        request.InfoSum.GetName(),
		Link:        request.InfoSum.GetLink(),
		Image:       request.InfoSum.GetImage(),
		Description: request.InfoSum.GetDescription(),
	})
	if err != nil {
		return nil, SAVE_INFOSUM_ERROR(err)
	}

	return &contentv1.SaveInfoSumResponse{}, nil
}

func (c *ContentServiceServer) DelInfoSum(ctx context.Context, request *contentv1.DelInfoSumRequest) (*contentv1.DelInfoSumResponse, error) {
	// 1. 执行删除
	err := c.svcInfoSum.Del(ctx, request.GetId())
	if err != nil {
		return nil, DEL_INFOSUM_ERROR(err)
	}

	return &contentv1.DelInfoSumResponse{}, nil
}

func infoSumDomains2GRPC(infos []domain.InfoSum) []*contentv1.InfoSum {
	res := make([]*contentv1.InfoSum, 0, len(infos))
	for _, i := range infos {
		res = append(res, &contentv1.InfoSum{
			Id:          int64(i.ID),
			Name:        i.Name,
			Link:        i.Link,
			Image:       i.Image,
			Description: i.Description,
		})
	}
	return res
}
