package user

import (
	"context"

	classlistv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/classlist/v1"
	feedv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feed/v1"
	gradev1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/grade/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/spf13/viper"
)

type PreLoader interface {
	PreLoad(ctx context.Context, studentId string)
}

func NewPreLoader(
	gradeClient gradev1.GradeServiceClient,
	classerClient classlistv1.ClasserClient,
	feedClient feedv1.FeedServiceClient,
	l logger.Logger,
) PreLoader {

	return &preLoader{
		gradeClient:     gradeClient,
		classerClient:   classerClient,
		feedClient:      feedClient,
		l:               l,
		currentSemester: viper.GetString("classlist.currentSemester"),
		currentYear:     viper.GetString("classlist.currentYear"),
	}
}

type preLoader struct {
	gradeClient     gradev1.GradeServiceClient
	classerClient   classlistv1.ClasserClient
	feedClient      feedv1.FeedServiceClient
	l               logger.Logger
	currentSemester string
	currentYear     string
}

func (l *preLoader) PreLoad(ctx context.Context, studentId string) {
	// 预创建feed的配置列表
	go func() {
		_, _ = l.feedClient.FindOrCreateAllowList(ctx, &feedv1.FindOrCreateAllowListReq{StudentId: studentId})
	}()

	// 异步获取学生成绩,不需要等待结果
	go func() {
		_, _ = l.gradeClient.GetGradeScore(ctx, &gradev1.GetGradeScoreReq{
			StudentId: studentId,
		})
	}()

	go func() {
		_, _ = l.gradeClient.GetGradeByTerm(ctx, &gradev1.GetGradeByTermReq{
			StudentId: studentId,
			Refresh:   true,
			Kcxzmcs:   []string{"1"},
		})
	}()

	// 异步获取学生课表,不需要等待结果
	go func() {
		_, _ = l.classerClient.GetClass(ctx, &classlistv1.GetClassRequest{
			Refresh:  true,
			StuId:    studentId,
			Year:     l.currentYear,
			Semester: l.currentSemester,
		})
	}()
}
