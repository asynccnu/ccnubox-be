package user

import (
	"context"
	gradev1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/grade/v1"
)

type GradeGetter interface {
	PreGetStudentGrade(ctx context.Context, studentId string)
}

type gradeGetter struct {
	gradeClient gradev1.GradeServiceClient
}

func NewGradeGetter(gradeClient gradev1.GradeServiceClient) GradeGetter {
	return &gradeGetter{gradeClient: gradeClient}
}

func (g *gradeGetter) PreGetStudentGrade(ctx context.Context, studentId string) {
	// 异步获取学生成绩,不需要等待结果
	go func() {
		_, _ = g.gradeClient.GetGradeScore(ctx, &gradev1.GetGradeScoreReq{
			StudentId: studentId,
		})
	}()

	go func() {
		_, _ = g.gradeClient.GetGradeByTerm(ctx, &gradev1.GetGradeByTermReq{
			StudentId: studentId,
			Refresh:   true,
			Kcxzmcs:   []string{"1"},
		})
	}()
}
