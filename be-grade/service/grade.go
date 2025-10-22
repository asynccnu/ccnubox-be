package service

import (
	"context"
	"errors"
	"time"

	"golang.org/x/sync/singleflight"

	gradev1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/grade/v1"
	userv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/user/v1"
	"github.com/asynccnu/ccnubox-be/be-grade/domain"
	"github.com/asynccnu/ccnubox-be/be-grade/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/be-grade/pkg/logger"
	"github.com/asynccnu/ccnubox-be/be-grade/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-grade/repository/model"
)

var (
	ErrGetGrade = func(err error) error {
		return errorx.New(gradev1.ErrorGetGradeError("获取成绩失败"), "dao", err)
	}
)

type GradeService interface {
	GetGradeByTerm(ctx context.Context, req *domain.GetGradeByTermReq) ([]domain.Grade, error)
	GetGradeScore(ctx context.Context, studentId string) ([]domain.TypeOfGradeScore, error)
	GetUpdateScore(ctx context.Context, studentId string) ([]domain.Grade, error)
	GetDistinctGradeType(ctx context.Context)([]string,error)
}

type gradeService struct {
	userClient userv1.UserServiceClient
	gradeDAO   dao.GradeDAO
	l          logger.Logger
	sf         singleflight.Group
}

func NewGradeService(gradeDAO dao.GradeDAO, l logger.Logger, userClient userv1.UserServiceClient) GradeService {
	return &gradeService{gradeDAO: gradeDAO, l: l, userClient: userClient}
}

func (s *gradeService) GetGradeByTerm(ctx context.Context, req *domain.GetGradeByTermReq) ([]domain.Grade, error) {
	grades, err := s.getGradeWithSingleFlight(ctx, req.StudentID, req.Refresh)
	if err != nil {
		return nil, err
	}
	return modelConvDomainAndFilter(grades, req.Terms, req.Kcxzmcs), nil
}

func (s *gradeService) GetGradeScore(ctx context.Context, studentId string) ([]domain.TypeOfGradeScore, error) {
	grades, err := s.getGradeWithSingleFlight(ctx, studentId, false)
	if err != nil {
		return nil, err
	}
	return aggregateGradeScore(grades), nil
}

func (s *gradeService) GetUpdateScore(ctx context.Context, studentId string) ([]domain.Grade, error) {
	grades, err := s.fetchGradesFromRemote(ctx, studentId)
	if err != nil || len(grades) == 0 {
		return nil, ErrGetGrade(err)
	}

	updated, err := s.gradeDAO.BatchInsertOrUpdate(context.Background(), grades)
	if err != nil {
		s.l.Warn("更新成绩失败", logger.Error(err))
		return nil, ErrGetGrade(err)
	}

	for _, g := range updated {
		s.l.Info("更新成绩成功", logger.String("studentId", g.Studentid), logger.String("课程", g.Kcmc))
	}
	return modelConvDomain(updated), nil
}

func (s *gradeService) getGradeWithSingleFlight(ctx context.Context, studentId string, refresh bool) ([]model.Grade, error) {
	if refresh {
		_, grades, err := s.fetchGradesFromRemoteAndUpdate(ctx, studentId, true)
		if err != nil || len(grades) == 0 {
			s.l.Warn("从ccnu获取成绩失败!", logger.Error(err))
			grades, err = s.gradeDAO.FindGrades(context.Background(), studentId, 0, 0)
			if err != nil {
				return nil, ErrGetGrade(err)
			}
		}
		return grades, nil
	}

	//从数据库获取数据
	grades, err := s.gradeDAO.FindGrades(ctx, studentId, 0, 0)
	if err == nil && len(grades) > 0 {
		//如果有成绩进行异步更新
		go func() {
			_, _, err := s.fetchGradesFromRemoteAndUpdate(context.Background(), studentId, true)
			if err != nil {
				s.l.Warn("从ccnu获取成绩失败!", logger.Error(err))
			}
		}()
		return grades, nil
	}

	//如果没成绩尝试获取最新成绩
	s.l.Info("数据库中无成绩或查询失败，尝试从ccnu获取", logger.Error(err))
	_, grades, err = s.fetchGradesFromRemoteAndUpdate(ctx, studentId, false)
	if err != nil {
		return nil, ErrGetGrade(err)
	}

	return grades, nil
}

func (s *gradeService) fetchGradesFromRemote(ctx context.Context, studentId string) ([]model.Grade, error) {
	key := studentId

	result, err, _ := s.sf.Do(key, func() (interface{}, error) {
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		start := time.Now()
		cookieResp, err := s.userClient.GetCookie(ctx, &userv1.GetCookieRequest{StudentId: studentId})
		if err != nil {
			return nil, err
		}

		var stu Student
		if isUndergraduate(studentId) {
			stu = &UndergraduateStudent{}
		} else {
			stu = &GraduateStudent{}
		}

		grades, err := stu.GetGrades(ctx, cookieResp.GetCookie(), 0, 0, 300)
		if errors.Is(err, COOKIE_TIMEOUT) {
			cookieResp, err = s.userClient.GetCookie(ctx, &userv1.GetCookieRequest{StudentId: studentId})
			if err != nil {
				return nil, err
			}
			return stu.GetGrades(ctx, cookieResp.GetCookie(), 0, 0, 300)
		}
		s.l.Info("获取成绩耗时", logger.String("耗时", time.Since(start).String()))

		return grades, err
	})

	grades, ok := result.([]model.Grade)
	if !ok {
		s.l.Warn("类型断言失败", logger.Error(err))
	}

	return grades, err

}

func (s *gradeService) updateGrades(grades []model.Grade) ([]model.Grade, error) {
	updated, err := s.gradeDAO.BatchInsertOrUpdate(context.Background(), grades)
	if err != nil {
		return nil, err
	}

	for _, g := range updated {
		s.l.Info("更新成绩成功", logger.String("studentId", g.Studentid), logger.String("课程", g.Kcmc))
	}
	return updated, nil
}

func (s *gradeService) fetchGradesFromRemoteAndUpdate(ctx context.Context, studentId string, isAsyc bool) (updated []model.Grade, grades []model.Grade, err error) {

	remote, err := s.fetchGradesFromRemote(ctx, studentId)
	if err != nil {
		return nil, nil, err
	}

	if isAsyc {
		go func() {
			_, err := s.updateGrades(remote)
			if err != nil {
				s.l.Warn("异步更新成绩失败", logger.Error(err))
			}
		}()
		return nil, remote, nil
	}

	update, err := s.updateGrades(remote)
	if err != nil {
		return nil, remote, err
	}

	return update, remote, nil
}

func (s *gradeService)GetDistinctGradeType(ctx context.Context)([]string,error){
	res,err:=s.gradeDAO.GetDistinctGradeType(ctx)
	if err!=nil{
		s.l.Warn("获取课程性质列表失败",logger.Error(err))
		return nil,err
	}
	return res,nil
}

type Student interface {
	GetGrades(ctx context.Context, cookie string, xnm, xqm, showCount int64) ([]model.Grade, error)
}

type UndergraduateStudent struct{}

func (u *UndergraduateStudent) GetGrades(ctx context.Context, cookie string, xnm, xqm, showCount int64) ([]model.Grade, error) {
	return GetGrade(ctx, cookie, xnm, xqm, showCount)
}

type GraduateStudent struct{}

func (g *GraduateStudent) GetGrades(ctx context.Context, cookie string, xnm, xqm, showCount int64) ([]model.Grade, error) {
	return GetGraduateGrades(ctx, cookie, xnm, xqm, showCount)
}

func isUndergraduate(stuID string) bool {
	if len(stuID) < 5 {
		return false
	}
	return stuID[4] == '2'
}
