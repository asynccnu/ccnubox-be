package service

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/asynccnu/ccnubox-be/be-grade/crawler"
	"github.com/asynccnu/ccnubox-be/be-grade/domain"
	"github.com/asynccnu/ccnubox-be/be-grade/events/producer"
	"github.com/asynccnu/ccnubox-be/be-grade/events/topic"
	"github.com/asynccnu/ccnubox-be/be-grade/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-grade/repository/model"
	classlistv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/classlist/v1"
	gradev1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/grade/v1"
	proxyv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/proxy/v1"
	userv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/user/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/robfig/cron/v3"
	"golang.org/x/sync/singleflight"
)

var (
	ErrGetGrade = errorx.FormatErrorFunc(gradev1.ErrorGetGradeError("获取成绩失败"))
)

// 创建一个全局client
var client = &http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse // 禁止自动跳转，返回原始响应
	},
	Transport: &http.Transport{
		MaxIdleConns:        100, // 最大空闲连接数
		MaxIdleConnsPerHost: 10,  // 每个主机最大空闲连接数
		MaxConnsPerHost:     100, // 每个主机最大连接数
	},
}

type GradeService interface {
	GetGradeByTerm(ctx context.Context, req *domain.GetGradeByTermReq) ([]domain.Grade, error)
	GetGradeScore(ctx context.Context, studentId string) ([]domain.TypeOfGradeScore, error)
	GetUpdateScore(ctx context.Context, studentId string) ([]domain.Grade, error)
	UpdateDetailScore(ctx context.Context, need domain.NeedDetailGrade) error
	GetDistinctGradeType(ctx context.Context, stuID string) ([]string, error)
}

type gradeService struct {
	userClient      userv1.UserServiceClient
	proxyClient     proxyv1.ProxyClient
	classlistClient classlistv1.ClasserClient
	gradeDAO        dao.GradeDAO
	l               logger.Logger
	sf              singleflight.Group
	producer        producer.Producer
}

func NewGradeService(gradeDAO dao.GradeDAO, l logger.Logger, userClient userv1.UserServiceClient, classlistClient classlistv1.ClasserClient, proxyClient proxyv1.ProxyClient, producer producer.Producer) GradeService {
	g := &gradeService{
		gradeDAO:        gradeDAO,
		l:               l,
		userClient:      userClient,
		proxyClient:     proxyClient,
		producer:        producer,
		classlistClient: classlistClient,
	}

	g.pullProxyAddr()
	beginCron(g)

	return g
}

func beginCron(s *gradeService) {
	cr := cron.New()
	_, _ = cr.AddFunc("@every 160s", s.pullProxyAddr)
	cr.Start()
}

func (s *gradeService) pullProxyAddr() {
	res, err := s.proxyClient.GetProxyAddr(context.Background(), &proxyv1.GetProxyAddrRequest{})
	if err != nil {
		s.l.Warn("service: get proxy addr failed", logger.Error(err))
		res = &proxyv1.GetProxyAddrResponse{Addr: ""}
	}

	if res.Addr == "" {
		return
	}

	proxy, err := url.Parse(res.Addr)
	if err != nil {
		s.l.Warn("service: parse proxy addr failed", logger.String("addr", res.Addr), logger.Error(err))
		return
	}

	p := http.ProxyURL(proxy)
	client.Transport.(*http.Transport).Proxy = p
}

func (s *gradeService) GetGradeByTerm(ctx context.Context, req *domain.GetGradeByTermReq) ([]domain.Grade, error) {
	var (
		grades    []model.Grade
		fetchdata FetchGrades
		err       error
	)
	l := s.l.WithContext(ctx)
	if req.Refresh {
		// 强制更新：拉取远程
		fetchdata, err = s.fetchGradesWithSingleFlight(ctx, req.StudentID)
		if err != nil || len(fetchdata.final) == 0 {
			l.Warn("service: force refresh failed, using local fallback", logger.String("sid", req.StudentID), logger.Error(err))
			// 拉取失败本地作为兜底
			grades, err = s.gradeDAO.FindGrades(ctx, req.StudentID, 0, 0)
			if err != nil {
				return nil, errorx.Errorf("service: fallback dao find failed, sid: %s, err: %w", req.StudentID, err)
			}
			return modelConvDomainAndFilter(grades, req.Terms, req.Kcxzmcs), nil
		}
		grades = fetchdata.final
		return modelConvDomainAndFilter(grades, req.Terms, req.Kcxzmcs), nil

	} else {
		// 优先查本地缓存
		grades, err = s.gradeDAO.FindGrades(ctx, req.StudentID, 0, 0)
		if err != nil || len(grades) == 0 {
			// 本地无数据，从远程拉取
			fetchdata, err = s.fetchGradesWithSingleFlight(ctx, req.StudentID)
			if err != nil {
				return nil, ErrGetGrade(errorx.Errorf("service: cache miss and fetch failed, sid: %s, err: %w", req.StudentID, err))
			}
			grades = fetchdata.final
			return modelConvDomainAndFilter(grades, req.Terms, req.Kcxzmcs), nil
		}

		// 本地有数据，异步触发一次更新
		go func() {
			_, _ = s.fetchGradesWithSingleFlight(context.Background(), req.StudentID)
		}()

		return modelConvDomainAndFilter(grades, req.Terms, req.Kcxzmcs), nil
	}
}

func (s *gradeService) GetGradeScore(ctx context.Context, studentId string) ([]domain.TypeOfGradeScore, error) {
	grades, err := s.gradeDAO.FindGrades(ctx, studentId, 0, 0)
	if err != nil || len(grades) == 0 {
		fetchdata, err := s.fetchGradesWithSingleFlight(ctx, studentId)
		if err != nil || len(fetchdata.final) == 0 {
			return nil, ErrGetGrade(errorx.Errorf("service: get grade score fetch failed, sid: %s, err: %w", studentId, err))
		}
		return aggregateGradeScore(fetchdata.final), nil
	}

	go func() {
		_, _ = s.fetchGradesWithSingleFlight(context.Background(), studentId)
	}()

	return aggregateGradeScore(grades), nil
}

func (s *gradeService) GetUpdateScore(ctx context.Context, studentId string) ([]domain.Grade, error) {
	grades, err := s.fetchGradesWithSingleFlight(ctx, studentId)
	if err != nil || len(grades.update) == 0 {
		return nil, ErrGetGrade(errorx.Errorf("service: get update score failed, sid: %s, err: %w", studentId, err))
	}
	return modelConvDomain(grades.update), nil
}

func (s *gradeService) UpdateDetailScore(ctx context.Context, need domain.NeedDetailGrade) error {
	l := s.l.WithContext(ctx)
	ug, err := s.newUGWithCookie(ctx, need.StudentID)
	if err != nil {
		return errorx.Errorf("service: init ug for detail failed, sid: %s, err: %w", need.StudentID, err)
	}

	grades := need.Grades
	for i, grade := range grades {
		detail, err := ug.GetDetail(ctx, grade.StudentId, grade.JxbId, grade.KcId, grade.Cj)
		if errors.Is(err, crawler.ErrCookieTimeout) {
			ug, err = s.newUGWithCookie(ctx, need.StudentID)
			if err != nil {
				return errorx.Errorf("service: refresh cookie for detail failed, sid: %s, err: %w", need.StudentID, err)
			}
			detail, err = ug.GetDetail(ctx, grade.StudentId, grade.JxbId, grade.KcId, grade.Cj)
		}

		if err != nil {
			l.Warn("service: fetch partial detail failed", logger.String("sid", grade.StudentId), logger.String("kc", grade.Kcmc), logger.Error(err))
			continue
		}

		if detail.Cjxm3 == 0 && detail.Cjxm1 == 0 && detail.Cjxm3bl != "" && detail.Cjxm1bl != "" {
			continue
		}

		grade.RegularGradePercent = detail.Cjxm3bl
		grade.RegularGrade = detail.Cjxm3
		grade.FinalGradePercent = detail.Cjxm1bl
		grade.FinalGrade = detail.Cjxm1
		grades[i] = grade
	}

	_, err = s.gradeDAO.BatchInsertOrUpdate(ctx, grades, true)
	if err != nil {
		return errorx.Errorf("service: batch save details failed, sid: %s, err: %w", need.StudentID, err)
	}

	return nil
}

func (s *gradeService) fetchGradesWithSingleFlight(ctx context.Context, studentId string) (FetchGrades, error) {
	l := s.l.WithContext(ctx)
	result, err, _ := s.sf.Do(studentId, func() (interface{}, error) {
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		var stu Student
		if isUndergraduate(studentId) {
			ug, err := s.newUGWithCookie(ctx, studentId)
			if err != nil {
				return nil, errorx.Errorf("service: init undergraduate crawler failed, err: %w", err)
			}
			stu = &UndergraduateStudent{ug: ug}
		} else {
			gc, err := crawler.NewGraduate(crawler.NewCrawlerClientWithCookieJar(30*time.Second, nil))
			if err != nil {
				return nil, errorx.Errorf("service: init graduate crawler failed, err: %w", err)
			}
			stu = &GraduateStudent{gc: gc}
		}

		cookieResp, err := s.userClient.GetCookie(ctx, &userv1.GetCookieRequest{StudentId: studentId})
		if err != nil {
			return nil, errorx.Errorf("service: rpc get cookie failed, sid: %s, err: %w", studentId, err)
		}

		remote, err := stu.GetGrades(ctx, cookieResp.Cookie, 0, 0, 300)
		if errors.Is(err, crawler.ErrCookieTimeout) {
			cookieResp, err = s.userClient.GetCookie(ctx, &userv1.GetCookieRequest{StudentId: studentId})
			if err != nil {
				return nil, errorx.Errorf("service: retry rpc get cookie failed, sid: %s, err: %w", studentId, err)
			}
			remote, err = stu.GetGrades(ctx, cookieResp.Cookie, 0, 0, 300)
		}

		if err != nil {
			return nil, errorx.Errorf("service: crawler fetch failed, sid: %s, err: %w", studentId, err)
		}

		update, err := s.gradeDAO.BatchInsertOrUpdate(ctx, remote, false)
		if err != nil {
			return nil, errorx.Errorf("service: dao batch sync failed, sid: %s, err: %w", studentId, err)
		}

		final, err := s.gradeDAO.FindGrades(ctx, studentId, 0, 0)
		if err != nil {
			return nil, errorx.Errorf("service: dao find final failed, sid: %s, err: %w", studentId, err)
		}

		// 异步获取详情的 MQ 触发
		var needDetailgrades []model.Grade
		for _, g := range final {
			if g.RegularGradePercent == RegularGradePercentMSG && g.FinalGradePercent == FinalGradePercentMAG {
				needDetailgrades = append(needDetailgrades, g)
			}
		}

		if len(needDetailgrades) > 0 {
			err = s.producer.SendMessage(topic.GradeDetailEvent, domain.NeedDetailGrade{
				StudentID: studentId,
				Grades:    needDetailgrades,
			})
			if err != nil {
				l.Error("service: send detail event failed", logger.String("sid", studentId), logger.Error(err))
			}
		}

		return FetchGrades{update: update, final: final}, nil
	})

	fetchGrades, ok := result.(FetchGrades)
	if !ok && err == nil {
		err = errorx.Errorf("service: fetch result type assertion failed")
	}

	return fetchGrades, err
}

func (s *gradeService) newUGWithCookie(ctx context.Context, studentId string) (*crawler.UnderGrad, error) {
	cookieResp, err := s.userClient.GetCookie(ctx, &userv1.GetCookieRequest{StudentId: studentId})
	if err != nil {
		return nil, errorx.Errorf("service: newUG rpc cookie failed, sid: %s, err: %w", studentId, err)
	}

	grad, err := crawler.NewUnderGrad(
		crawler.NewCrawlerClientWithCookieJar(
			30*time.Second,
			crawler.NewJarWithCookie(crawler.PG_URL, cookieResp.Cookie),
		),
	)
	if err != nil {
		return nil, errorx.Errorf("service: newUnderGrad crawler init failed, err: %w", err)
	}
	return grad, nil
}

func (s *gradeService) GetDistinctGradeType(ctx context.Context, stuID string) ([]string, error) {
	l := s.l.WithContext(ctx)
	var res []string
	resp, err := s.classlistClient.GetClassNatures(ctx, &classlistv1.GetClassNaturesReq{StuId: stuID})
	if err != nil {
		l.Warn("service: rpc get class natures failed", logger.String("sid", stuID), logger.Error(err))
	}

	localNatures, err := s.gradeDAO.GetDistinctGradeType(ctx, stuID)
	if err != nil {
		return nil, errorx.Errorf("service: dao get distinct grade type failed, sid: %s, err: %w", stuID, err)
	}

	natureSet := make(map[string]struct{})
	if resp != nil {
		for _, nature := range resp.ClassNatures {
			natureSet[nature] = struct{}{}
		}
	}
	for _, nature := range localNatures {
		natureSet[nature] = struct{}{}
	}
	for nature := range natureSet {
		res = append(res, nature)
	}

	return res, nil
}

// Student 接口及实现
type Student interface {
	GetGrades(ctx context.Context, cookie string, xnm, xqm, showCount int64) ([]model.Grade, error)
}

type UndergraduateStudent struct {
	ug *crawler.UnderGrad
}

func (u *UndergraduateStudent) GetGrades(ctx context.Context, cookie string, xnm, xqm, showCount int64) ([]model.Grade, error) {
	grade, err := u.ug.GetGrade(ctx, xnm, xqm, int(showCount))
	if err != nil {
		return nil, errorx.Errorf("crawler: ug get grade failed, err: %w", err)
	}

	details := make(map[string]crawler.Score)
	for _, g := range grade {
		detail, err := u.ug.GetDetail(ctx, g.XS0101ID, g.JX0404ID, g.KCH, g.ZCJ)
		if err != nil {
			return nil, errorx.Errorf("crawler: ug get detail failed, jxb: %s, err: %w", g.JX0404ID, err)
		}
		key := g.XS0101ID + g.JX0404ID
		details[key] = detail
	}

	return aggregateGrade(grade, details), nil
}

type GraduateStudent struct {
	gc *crawler.Graduate
}

func (g *GraduateStudent) GetGrades(ctx context.Context, cookie string, xnm, xqm, showCount int64) ([]model.Grade, error) {
	grade, err := g.gc.GetGraduateGrades(ctx, cookie, xnm, xqm, showCount)
	if err != nil {
		return nil, errorx.Errorf("crawler: graduate get grade failed, err: %w", err)
	}

	return ConvertGraduateGrade(grade), nil
}

func isUndergraduate(stuID string) bool {
	if len(stuID) < 5 {
		return false
	}
	return stuID[4] == '2'
}
