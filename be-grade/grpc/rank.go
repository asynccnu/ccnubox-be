package grpc

import (
	"context"

	v1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/grade/v1"
	"github.com/asynccnu/ccnubox-be/be-grade/domain"
)

func (s *GradeServiceServer) GetRankByTerm(ctx context.Context, req *v1.GetRankByTermReq) (*v1.GetRankByTermResp, error) {
	data, err := s.rankSer.GetRankByTerm(ctx, convGetRankByTermReqReqFromProtoToDomain(req))
	if err != nil {
		return nil, err
	}

	return &v1.GetRankByTermResp{
		Rank:    data.Rank,
		Score:   data.Score,
		Include: data.Include,
	}, nil

}

func (s *GradeServiceServer) LoadRank(ctx context.Context, req *v1.LoadRankReq) (*v1.EmptyResp, error) {
	s.rankSer.LoadRank(ctx, convLoadRankReqFromProtoToDomain(req))

	return nil, nil
}

func convGetRankByTermReqReqFromProtoToDomain(req *v1.GetRankByTermReq) *domain.GetRankByTermReq {
	if req == nil {
		return nil
	}

	return &domain.GetRankByTermReq{
		StudentId: req.StudentId,
		XnmBegin:  req.XnmBegin,
		XqmBegin:  req.XqmBegin,
		XnmEnd:    req.XnmEnd,
		XqmEnd:    req.XqmEnd,
		Refresh:   req.Refresh,
	}
}

func convLoadRankReqFromProtoToDomain(req *v1.LoadRankReq) *domain.LoadRankReq {
	if req == nil {
		return nil
	}

	return &domain.LoadRankReq{
		StudentId: req.StudentId,
	}
}
