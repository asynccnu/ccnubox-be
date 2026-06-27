package cron

import (
	"context"
	"fmt"
	"net/url"
	"strings"

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
	l logger.Logger) tieredx.RefreshHandler {
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

func getGradeEventUrl(semester string, courseType string) string {
	values := url.Values{}
	values.Add("semester", fmt.Sprintf(`["%s"]`, semester))
	values.Add("courseType", fmt.Sprintf(`["%s"]`, courseType))
	return fmt.Sprintf("ccnubox://scoreCalculation?%s", values.Encode())
}

func formatSemester(xnm int64, xqm int64) string {
	return fmt.Sprintf("%d-%d", xnm, xqm)
}

func (p *TieredHandler) gradeRefresh(ctx context.Context, studentId string) error {
	//获取学生最新成绩
	resp, err := p.gradeClient.GetUpdateScore(ctx, &gradev1.GetUpdateScoreReq{StudentId: studentId})
	if err != nil {
		return err
	}

	for _, g := range resp.GetGrades() {
		//找出相同教学班的同学
		res, err := p.classList.GetStuIdByJxbId(ctx, &classlistv1.GetStuIdByJxbIdRequest{JxbId: g.JxbId})
		if err != nil {
			p.l.Error("获取学生ID失败", logger.Error(err), logger.String("JxbId", g.JxbId))
			continue
		}

		//把等级提到最高
		_, err = p.counter.ChangeCounterLevels(ctx, &counterv1.ChangeCounterLevelsReq{
			StudentIds: res.StuId,
			IsReduce:   false,
			Step:       int64(counterv1.CounterLevel_LEVEL_THERE),
		})
		if err != nil {
			p.l.Error("更改优先级发生错误", logger.Error(err))
			continue
		}

		semester := formatSemester(g.Xnm, g.Xqm)
		url := getGradeEventUrl(semester, g.Kcxzmc)
		//成绩更新消息推送
		_, err = p.feedClient.PublicFeedEvent(ctx, &feedv1.PublicFeedEventReq{
			StudentId: studentId,
			IsAll:     false,
			Event: &feedv1.FeedEvent{
				Type:         feedv1.FeedEventType_GRADE,
				Title:        "成绩更新提醒",
				Content:      fmt.Sprintf("您的课程:%s分数更新了,请及时查看", g.Kcmc),
				Url:          url,
				ExtendFields: map[string]string{"url": url},
			},
		})
		if err != nil {
			p.l.Error("推送错误", logger.Error(err))
		}
	}

	return nil

}

func (p *TieredHandler) classListRefresh(ctx context.Context, studentId string) error {
	//判断当前学期
	res, err := p.content.GetSemester(ctx, &contentv1.GetSemesterRequest{})
	if err != nil {
		p.l.Errorf("student:%s get current semester error：%v", studentId, err)
		return err
	}
	//把学期字符串解析成year和semester
	strs := strings.Split(res.Semester, "-")
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
