package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/asynccnu/ccnubox-be/be-grade/crawler"
	"github.com/asynccnu/ccnubox-be/be-grade/events/producer"
	"github.com/asynccnu/ccnubox-be/be-grade/events/topic"
	proxyv1 "github.com/asynccnu/ccnubox-be/common/be-api/gen/proto/proxy/v1"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/robfig/cron/v3"

	"golang.org/x/sync/singleflight"

	"github.com/asynccnu/ccnubox-be/be-grade/domain"
	"github.com/asynccnu/ccnubox-be/be-grade/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-grade/repository/model"
	gradev1 "github.com/asynccnu/ccnubox-be/common/be-api/gen/proto/grade/v1"
	userv1 "github.com/asynccnu/ccnubox-be/common/be-api/gen/proto/user/v1"
	errorx "github.com/asynccnu/ccnubox-be/common/pkg/errorx/rpcerr"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

var (
	ErrGetGrade = func(err error) error {
		return errorx.New(gradev1.ErrorGetGradeError("获取成绩失败"), "data", err)
	}
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
	userClient  userv1.UserServiceClient
	proxyClient proxyv1.ProxyClient
	gradeDAO    dao.GradeDAO
	l           logger.Logger
	sf          singleflight.Group
	producer    producer.Producer
}

func NewGradeService(gradeDAO dao.GradeDAO, l logger.Logger, userClient userv1.UserServiceClient, proxyClient proxyv1.ProxyClient, producer producer.Producer) GradeService {
	g := &gradeService{gradeDAO: gradeDAO, l: l, userClient: userClient, proxyClient: proxyClient, producer: producer}

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
		log.Warn("get proxy addr err: ", err)
		res = &proxyv1.GetProxyAddrResponse{Addr: ""}
	}

	proxy, err := url.Parse(res.Addr)
	if err != nil {
		log.Warn("parse proxy addr err: ", err)
	}

	p := http.ProxyURL(proxy)

	client.Transport.(*http.Transport).Proxy = p
	log.Debug("update client proxy addr, now: ", time.Now())
}

func (s *gradeService) GetGradeByTerm(ctx context.Context, req *domain.GetGradeByTermReq) ([]domain.Grade, error) {
	var (
		grades    []model.Grade
		fetchdata FetchGrades
		err       error
	)

	if req.Refresh {

		//如果要求强制更新的话就需要去拉取远程数据
		fetchdata, err = s.fetchGradesWithSingleFlight(ctx, req.StudentID)
		if err != nil || len(fetchdata.final) == 0 {
			s.l.Warn("从ccnu获取成绩失败!", logger.Error(err))
			//拉取失败本地作为兜底
			grades, err = s.gradeDAO.FindGrades(ctx, req.StudentID, 0, 0)
			if err != nil {
				return nil, err
			}
			return modelConvDomainAndFilter(grades, req.Terms, req.Kcxzmcs), nil
		}

		grades = fetchdata.final
		return modelConvDomainAndFilter(grades, req.Terms, req.Kcxzmcs), nil

	} else {

		//如果成功直接返回结果
		grades, err = s.gradeDAO.FindGrades(ctx, req.StudentID, 0, 0)
		if err != nil || len(grades) == 0 {
			//失败尝试从远程拉取
			fetchdata, err = s.fetchGradesWithSingleFlight(ctx, req.StudentID)
			if err != nil {
				s.l.Warn("从ccnu获取成绩失败!", logger.Error(err))
				return nil, ErrGetGrade(err)
			}
			grades = fetchdata.final

			return modelConvDomainAndFilter(grades, req.Terms, req.Kcxzmcs), nil
		}

		//异步更新结果
		go func() {
			fetchdata, err = s.fetchGradesWithSingleFlight(context.Background(), req.StudentID)
			if err != nil {
				s.l.Warn("从ccnu获取成绩失败!", logger.Error(err))
			}
		}()

		return modelConvDomainAndFilter(grades, req.Terms, req.Kcxzmcs), nil
	}
}

func (s *gradeService) GetGradeScore(ctx context.Context, studentId string) ([]domain.TypeOfGradeScore, error) {
	//如果成功直接返回结果
	grades, err := s.gradeDAO.FindGrades(ctx, studentId, 0, 0)
	if err != nil || len(grades) == 0 {
		//失败尝试从远程拉取
		fetchdata, err := s.fetchGradesWithSingleFlight(ctx, studentId)
		if err != nil || len(fetchdata.final) == 0 {
			s.l.Warn("从ccnu获取成绩失败!", logger.Error(err))
			return nil, ErrGetGrade(err)
		}
		return aggregateGradeScore(fetchdata.final), nil
	}

	//异步更新结果
	go func() {
		fetchdata, err := s.fetchGradesWithSingleFlight(context.Background(), studentId)
		if err != nil || len(fetchdata.final) == 0 {
			s.l.Warn("从ccnu获取成绩失败!", logger.Error(err))
		}
	}()

	return aggregateGradeScore(grades), nil
}

func (s *gradeService) GetUpdateScore(ctx context.Context, studentId string) ([]domain.Grade, error) {
	grades, err := s.fetchGradesWithSingleFlight(ctx, studentId)
	if err != nil || len(grades.update) == 0 {
		return nil, ErrGetGrade(err)
	}
	return modelConvDomain(grades.update), nil
}

func (s *gradeService) UpdateDetailScore(ctx context.Context, need domain.NeedDetailGrade) error {
	ug, err := s.newUGWithCookie(ctx, need.StudentID)
	if err != nil {
		return err
	}

	grades := need.Grades
	for i, grade := range grades {
		detail, err := ug.GetDetail(ctx, grade.StudentId, grade.JxbId, grade.KcId, grade.Cj)
		if errors.Is(err, crawler.COOKIE_TIMEOUT) {
			ug, err = s.newUGWithCookie(ctx, need.StudentID)
			if err != nil {
				return err
			}
			detail, err = ug.GetDetail(ctx, grade.StudentId, grade.JxbId, grade.KcId, grade.Cj)
		}

		if err != nil {
			s.l.Warn(fmt.Sprintf("获取详细分数失败! 学号:%s,教学班id:%s,课程id:%s,总分:%f", grade.StudentId, grade.JxbId, grade.KcId, grade.Cj), logger.Error(err))
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
		return err
	}

	return nil
}

func (s *gradeService) fetchGradesWithSingleFlight(ctx context.Context, studentId string) (FetchGrades, error) {
	key := studentId

	result, err, _ := s.sf.Do(key, func() (interface{}, error) {
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		start := time.Now()
		// 按学号选择对应爬虫
		var stu Student
		if isUndergraduate(studentId) {
			ug, err := s.newUGWithCookie(ctx, studentId)
			if err != nil {
				return nil, fmt.Errorf("创建带cookie的本科爬虫失败:%w", err)
			}
			stu = &UndergraduateStudent{ug: ug}
		} else {
			gc, err := crawler.NewGraduate(crawler.NewCrawlerClientWithCookieJar(30*time.Second, nil))
			if err != nil {
				return nil, fmt.Errorf("创建研究生爬虫实例失败:%w", err)
			}
			stu = &GraduateStudent{gc: gc}
		}

		cookieResp, err := s.userClient.GetCookie(ctx, &userv1.GetCookieRequest{StudentId: studentId})
		if err != nil {
			return nil, fmt.Errorf("获取 cookie 失败:%w", err)
		}

		remote, err := stu.GetGrades(ctx, cookieResp.Cookie, 0, 0, 300)
		if errors.Is(err, crawler.COOKIE_TIMEOUT) {
			if _, ok := stu.(*UndergraduateStudent); ok {
				ug, err := s.newUGWithCookie(ctx, studentId)
				if err != nil {
					return nil, fmt.Errorf("创建带cookie的本科爬虫失败:%w", err)
				}
				stu = &UndergraduateStudent{ug: ug}
			}
			cookieResp, err = s.userClient.GetCookie(ctx, &userv1.GetCookieRequest{StudentId: studentId})
			if err != nil {
				return nil, fmt.Errorf("获取 cookie 失败:%w", err)
			}
			remote, err = stu.GetGrades(ctx, cookieResp.Cookie, 0, 0, 300)
			if err != nil {
				return nil, err
			}
		}

		s.l.Info("获取成绩耗时", logger.String("耗时", time.Since(start).String()))
		grades := remote
		update, err := s.gradeDAO.BatchInsertOrUpdate(ctx, grades, false)
		if err != nil {
			return nil, err
		}

		for _, g := range update {
			s.l.Info("更新成绩成功", logger.String("studentId", g.StudentId), logger.String("课程", g.Kcmc))
		}

		// 读取数据库,已经有的数据要使用已经存在的(因为平时成绩已经获取到了)
		final, err := s.gradeDAO.FindGrades(ctx, studentId, 0, 0)
		if err != nil {
			return nil, err
		}

		var needDetailgrades []model.Grade
		for _, g := range final {
			if g.RegularGradePercent == RegularGradePercentMSG && g.FinalGradePercent == FinalGradePercentMAG {
				needDetailgrades = append(needDetailgrades, g)
			}
		}

		err = s.producer.SendMessage(topic.GradeDetailEvent, domain.NeedDetailGrade{
			StudentID: studentId,
			Grades:    needDetailgrades,
		})
		if err != nil {
			return nil, err
		}

		fetchGrades := FetchGrades{
			update: update,
			final:  final,
		}

		return fetchGrades, err
	})

	fetchGrades, ok := result.(FetchGrades)
	if !ok {
		s.l.Warn("类型断言失败", logger.Error(err))
	}

	return fetchGrades, err
}

func (s *gradeService) newUGWithCookie(ctx context.Context, studentId string) (*crawler.UnderGrad, error) {
	cookieResp, err := s.userClient.GetCookie(ctx, &userv1.GetCookieRequest{StudentId: studentId})
	if err != nil {
		return &crawler.UnderGrad{}, err
	}

	grad, err := crawler.NewUnderGrad(
		crawler.NewCrawlerClientWithCookieJar(
			30*time.Second,
			crawler.NewJarWithCookie(crawler.PG_URL, cookieResp.Cookie),
		),
	)
	if err != nil {
		return &crawler.UnderGrad{}, fmt.Errorf("创建ug爬虫实例失败:%w", err)
	}
	return grad, nil
}

func (s *gradeService) GetDistinctGradeType(ctx context.Context, stuID string) ([]string, error) {
	res, err := s.gradeDAO.GetDistinctGradeType(ctx, stuID)
	if err != nil {
		s.l.Warn("获取课程性质列表失败", logger.Error(err))
		return nil, err
	}
	return res, nil
}

type Student interface {
	GetGrades(ctx context.Context, cookie string, xnm, xqm, showCount int64) ([]model.Grade, error)
}

type UndergraduateStudent struct {
	ug *crawler.UnderGrad
}

func (u *UndergraduateStudent) GetGrades(ctx context.Context, cookie string, xnm, xqm, showCount int64) ([]model.Grade, error) {
	grade, err := u.ug.GetGrade(ctx, xnm, xqm, int(showCount))
	if err != nil {
		return []model.Grade{}, err
	}

	details := make(map[string]crawler.Score)
	for _, g := range grade {
		detail, err := u.ug.GetDetail(ctx, g.XS0101ID, g.JX0404ID, g.KCH, g.ZCJ)
		if err != nil {
			return []model.Grade{}, err
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
		return []model.Grade{}, err
	}

	return ConvertGraduateGrade(grade), nil
}

func isUndergraduate(stuID string) bool {
	if len(stuID) < 5 {
		return false
	}
	return stuID[4] == '2'
}
