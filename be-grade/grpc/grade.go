package grpc

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-grade/domain"
	"github.com/asynccnu/ccnubox-be/be-grade/service"
	gradev1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/grade/v1"
	"google.golang.org/grpc"
)

type GradeServiceServer struct {
	gradev1.UnimplementedGradeServiceServer
	ser     service.GradeService
	rankSer service.RankService // 具体见 rank.go
}

func NewGradeGrpcService(ser service.GradeService, ser2 service.RankService) *GradeServiceServer {
	return &GradeServiceServer{ser: ser, rankSer: ser2}
}

func (s *GradeServiceServer) Register(server grpc.ServiceRegistrar) {
	gradev1.RegisterGradeServiceServer(server, s)
}

func (s *GradeServiceServer) GetGradeByTerm(ctx context.Context, req *gradev1.GetGradeByTermReq) (*gradev1.GetGradeByTermResp, error) {
	// 调用服务层获取成绩数据
	grades, err := s.ser.GetGradeByTerm(ctx, convGetGradeByTermReqFromProtoToDomain(req))
	if err != nil {
		return nil, err
	}

	// 初始化响应结构
	var resp gradev1.GetGradeByTermResp

	// 遍历数据库获取的成绩数据，逐一转化为 v1.Grade
	for _, g := range grades {
		// 将数据库模型中的字段映射到 Protobuf 的 v1.Grade 中
		resp.Grades = append(resp.Grades, convGradesFromDomainToProto(g))
	}

	// 返回填充后的响应
	return &resp, nil
}

func (s *GradeServiceServer) GetGradeScore(ctx context.Context, req *gradev1.GetGradeScoreReq) (*gradev1.GetGradeScoreResp, error) {
	scores, err := s.ser.GetGradeScore(ctx, req.GetStudentId())
	if err != nil {
		return nil, err
	}

	// 类型转换(grpc的类型转换真的很费劲)
	typeOfGradeScores := make([]*gradev1.TypeOfGradeScore, len(scores))
	for i, score := range scores {
		gradeScores := make([]*gradev1.GradeScore, len(score.GradeScoreList))

		for i := range score.GradeScoreList {
			gradeScores[i] = &gradev1.GradeScore{
				Kcmc: score.GradeScoreList[i].Kcmc,
				Xf:   score.GradeScoreList[i].Xf,
			}
		}

		typeOfGradeScores[i] = &gradev1.TypeOfGradeScore{
			Kcxzmc:         score.Kcxzmc,
			GradeScoreList: gradeScores,
		}

	}

	return &gradev1.GetGradeScoreResp{TypeOfGradeScore: typeOfGradeScores}, nil
}

func (s *GradeServiceServer) GetGradeType(ctx context.Context, req *gradev1.GetGradeTypeReq) (*gradev1.GetGradeTypeResp, error) {
	list, err := s.ser.GetDistinctGradeType(ctx, req.GetStudentId())
	if err != nil {
		return nil, err
	}
	return &gradev1.GetGradeTypeResp{
		GradeTypes: list,
	}, nil
}

func (s *GradeServiceServer) GetUpdateScore(ctx context.Context, in *gradev1.GetUpdateScoreReq) (*gradev1.GetUpdateScoreResp, error) {
	grades, err := s.ser.GetUpdateScore(ctx, in.GetStudentId())
	if err != nil {
		return nil, err
	}

	var resp gradev1.GetUpdateScoreResp

	for _, g := range grades {
		resp.Grades = append(resp.Grades, convGradesFromDomainToProto(g))
	}

	return &resp, nil
}

func convGetGradeByTermReqFromProtoToDomain(req *gradev1.GetGradeByTermReq) *domain.GetGradeByTermReq {
	if req == nil {
		return nil
	}

	terms := make([]domain.Term, 0, len(req.Terms))
	for _, t := range req.Terms {
		terms = append(terms, domain.Term{
			Xnm:  t.Xnm,
			Xqms: t.Xqms,
		})
	}

	return &domain.GetGradeByTermReq{
		StudentID: req.StudentId,
		Terms:     terms,
		Kcxzmcs:   req.Kcxzmcs,
		Refresh:   req.Refresh,
	}
}

func convGradesFromDomainToProto(g domain.Grade) *gradev1.Grade {
	return &gradev1.Grade{
		Xnm:                 g.Xnm,
		Xqm:                 g.Xqm,
		KcId:                g.KcId,
		JxbId:               g.JxbId,
		Kcmc:                g.Kcmc,
		Xf:                  g.Xf,
		Cj:                  g.Cj,
		Kcxzmc:              g.Kcxzmc,
		Kclbmc:              g.Kclbmc,
		Kcbj:                g.Kcbj,
		Jd:                  g.Jd,
		RegularGradePercent: g.RegularGradePercent,
		RegularGrade:        g.RegularGrade,
		FinalGradePercent:   g.FinalGradePercent,
		FinalGrade:          g.FinalGrade,
	}
}
