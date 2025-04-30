package grpc

import (
	"context"
	v1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/grade/v1"
	"github.com/asynccnu/ccnubox-be/be-grade/domain"
	"github.com/asynccnu/ccnubox-be/be-grade/service"
	"google.golang.org/grpc"
)

type GradeServiceServer struct {
	v1.UnimplementedGradeServiceServer
	ser service.GradeService
}

func NewGradeGrpcService(ser service.GradeService) *GradeServiceServer {
	return &GradeServiceServer{ser: ser}
}

func (s *GradeServiceServer) Register(server grpc.ServiceRegistrar) {
	v1.RegisterGradeServiceServer(server, s)
}

func (s *GradeServiceServer) GetGradeByTerm(ctx context.Context, req *v1.GetGradeByTermReq) (*v1.GetGradeByTermResp, error) {
	// 调用服务层获取成绩数据
	grades, err := s.ser.GetGradeByTerm(ctx, convGetGradeByTermReqFromProtoToDomain(req))
	if err != nil {
		return nil, err
	}

	// 初始化响应结构
	var resp v1.GetGradeByTermResp

	// 遍历数据库获取的成绩数据，逐一转化为 v1.Grade
	for _, g := range grades {
		// 将数据库模型中的字段映射到 Protobuf 的 v1.Grade 中
		resp.Grades = append(resp.Grades, &v1.Grade{
			Xnm:                 g.Xnm,
			Xqm:                 g.Xqm,
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
		})
	}

	// 返回填充后的响应
	return &resp, nil
}

func (s *GradeServiceServer) GetGradeScore(ctx context.Context, req *v1.GetGradeScoreReq) (*v1.GetGradeScoreResp, error) {
	scores, err := s.ser.GetGradeScore(ctx, req.GetStudentId())
	if err != nil {
		return nil, err
	}

	// 类型转换(grpc的类型转换真的很费劲)
	typeOfGradeScores := make([]*v1.TypeOfGradeScore, len(scores))
	for i, score := range scores {
		gradeScores := make([]*v1.GradeScore, len(score.GradeScoreList))

		for i := range score.GradeScoreList {
			gradeScores[i] = &v1.GradeScore{
				Kcmc: score.GradeScoreList[i].Kcmc,
				Xf:   score.GradeScoreList[i].Xf,
			}
		}

		typeOfGradeScores[i] = &v1.TypeOfGradeScore{
			Kcxzmc:         score.Kcxzmc,
			GradeScoreList: gradeScores,
		}

	}

	return &v1.GetGradeScoreResp{TypeOfGradeScore: typeOfGradeScores}, nil
}

func convGetGradeByTermReqFromProtoToDomain(req *v1.GetGradeByTermReq) *domain.GetGradeByTermReq {
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
