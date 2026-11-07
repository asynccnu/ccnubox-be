package cron

import (
	"context"
	"fmt"
	"strings"
	"time"

	classlistv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/classlist/v1"
	contentv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/content/v1"
	counterv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/counter/v1"
	feedv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feed/v1"
	gradev1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/grade/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/asynccnu/ccnubox-be/common/pkg/tieredx"
)

type TieredHandler struct {
	classList   classlistv1.ClasserClient
	gradeClient gradev1.GradeServiceClient
	feedClient  feedv1.FeedServiceClient
	content     contentv1.ContentServiceClient
	counter     counterv1.CounterServiceClient
	l           logger.Logger
}

func NewTieredHandler(classList classlistv1.ClasserClient,
	gradeClient gradev1.GradeServiceClient,
	feedClient feedv1.FeedServiceClient,
	content contentv1.ContentServiceClient,
	counter counterv1.CounterServiceClient,
	l logger.Logger,
) tieredx.RefreshHandler {
	return &TieredHandler{
		classList:   classList,
		gradeClient: gradeClient,
		feedClient:  feedClient,
		content:     content,
		counter:     counter,
		l:           l,
	}
}

func (p *TieredHandler) Refresh(ctx context.Context, studentId string) error {
	p.l.Infof("refresh")
	var errs []error
	err1 := p.gradeRefresh(ctx, studentId)
	if err1 != nil {
		p.l.Errorf("student:%s refresh grade error:%v", studentId, err1)
		errs = append(errs, err1)
	}

	err2 := p.classListRefresh(ctx, studentId)
	if err2 != nil {
		p.l.Errorf("student:%s refresh classlist error:%v", studentId, err2)
		errs = append(errs, err2)
	}

	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

func (p *TieredHandler) gradeRefresh(ctx context.Context, studentId string) error {
	// 通知已在 Grade 成绩事务中持久化，由 Outbox 唯一发布；BFF 只负责触发刷新和提速。
	resp, err := p.gradeClient.GetUpdateScore(ctx, &gradev1.GetUpdateScoreReq{StudentId: studentId})
	if err != nil {
		return err
	}
	p.boostGradeClassmates(ctx, resp.GetGrades())
	return nil
}

func (p *TieredHandler) boostGradeClassmates(ctx context.Context, grades []*gradev1.Grade) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	seen := make(map[string]struct{}, len(grades))
	for _, g := range grades {
		if g == nil || strings.TrimSpace(g.JxbId) == "" || g.JxbId == fmt.Sprintf("%s%d%d", g.Kcmc, g.Xnm, g.Xqm) {
			// 兼容 Grade DAO 的课程名+学年学期兜底 ID，这类 ID 不能用于查同班成员。
			continue
		}
		if _, ok := seen[g.JxbId]; ok {
			continue
		}
		seen[g.JxbId] = struct{}{}
		if ctx.Err() != nil {
			p.l.WithContext(ctx).Warn("成绩同班提速已取消或超时", logger.Error(ctx.Err()))
			return
		}
		res, err := p.classList.GetStuIdByJxbId(ctx, &classlistv1.GetStuIdByJxbIdRequest{JxbId: g.JxbId})
		if err != nil {
			p.l.WithContext(ctx).Error("获取学生ID失败", logger.Error(err), logger.String("JxbId", g.JxbId))
			continue
		}
		if len(res.GetStuId()) == 0 {
			continue
		}
		if _, err = p.counter.BoostScores(ctx, &counterv1.BoostScoresReq{StudentIds: res.GetStuId()}); err != nil {
			p.l.WithContext(ctx).Error("更改优先级发生错误", logger.String("JxbId", g.JxbId), logger.Error(err))
		}
	}
}

func (p *TieredHandler) classListRefresh(ctx context.Context, studentId string) error {
	// 判断当前学期
	res, err := p.content.GetSemester(ctx, &contentv1.GetSemesterRequest{})
	if err != nil {
		p.l.Errorf("student:%s get current semester error：%v", studentId, err)
		return err
	}
	// 把学期字符串解析成year和semester
	strs := strings.Split(res.Semester.Semester, "-")
	if len(strs) < 2 {
		p.l.Errorf("semester string is not valid")
		return errorx.Errorf("semester string is not valid")
	}
	year := strs[0]
	semester := strs[1]

	_, err = p.classList.GetClass(ctx, &classlistv1.GetClassRequest{
		StuId:    studentId,
		Semester: semester,
		Year:     year,
		Refresh:  true,
	})
	if err != nil {
		p.l.Errorf("student:%s refresh classlist err：%v", studentId, err)
		return err
	}

	return nil
}
