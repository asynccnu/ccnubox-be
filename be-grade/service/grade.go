package service

import (
	"context"
	gradev1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/grade/v1"
	userv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/user/v1"
	"github.com/asynccnu/ccnubox-be/be-grade/domain"
	"github.com/asynccnu/ccnubox-be/be-grade/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/be-grade/pkg/logger"
	"github.com/asynccnu/ccnubox-be/be-grade/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-grade/repository/model"
	"time"
)

var (
	GET_GRADE_ERROR = func(err error) error {
		return errorx.New(gradev1.ErrorGetGradeError("获取成绩失败"), "dao", err)
	}
)

type GradeService interface {
	GetGradeByTerm(ctx context.Context, req *domain.GetGradeByTermReq) ([]domain.Grade, error)
	GetGradeScore(ctx context.Context, StudentId string) ([]domain.TypeOfGradeScore, error)
	GetUpdateScore(ctx context.Context, studentId string) ([]domain.Grade, error)
}

type gradeService struct {
	userClient userv1.UserServiceClient
	gradeDAO   dao.GradeDAO
	l          logger.Logger
}

func NewGradeService(gradeDAO dao.GradeDAO, l logger.Logger, userClient userv1.UserServiceClient) GradeService {
	return &gradeService{gradeDAO: gradeDAO, l: l, userClient: userClient}
}

func (s *gradeService) GetGradeByTerm(ctx context.Context, req *domain.GetGradeByTermReq) ([]domain.Grade, error) {

	if req.Refresh {
		//如果强制刷新则尝试从ccnu获取数据
		grades, err := s.getGradeFromCCNU(ctx, req.StudentID)
		if len(grades) == 0 || err != nil {
			//记录日志
			s.l.Info("从ccnu获取成绩失败!", logger.FormatLog("ccnu", err)...)
			//尝试获取成绩
			grades, err = s.gradeDAO.FindGrades(context.Background(), req.StudentID, 0, 0)
			if err != nil {
				return nil, GET_GRADE_ERROR(err)
			}
			return modelConvDomainAndFilter(grades, req.Terms, req.Kcxzmcs), nil
		}

		// 异步更新成绩
		go func() {
			err := s.updateGrades(grades)
			if err != nil {
				s.l.Info("更新成绩失败!", logger.FormatLog("ccnu", err)...)
			}
		}()

		return modelConvDomainAndFilter(grades, req.Terms, req.Kcxzmcs), nil
	}

	//尝试从数据库获取成绩
	grades, err := s.gradeDAO.FindGrades(ctx, req.StudentID, 0, 0)
	if len(grades) == 0 || err != nil {

		s.l.Info("从数据库获取成绩失败!", logger.FormatLog("dao", err)...)

		//如果数据库没有查询到则尝试从ccnu获取数据
		grades, err = s.getGradeFromCCNU(ctx, req.StudentID)
		if len(grades) == 0 && err != nil {
			//ccnu这里如果是一些非法数据除非出错了否则不会爆error
			return nil, GET_GRADE_ERROR(err)
		}

		err := s.updateGrades(grades)
		if err != nil {
			s.l.Info("更新成绩失败!", logger.FormatLog("ccnu", err)...)
			return modelConvDomainAndFilter(grades, req.Terms, req.Kcxzmcs), nil
		}

		return modelConvDomainAndFilter(grades, req.Terms, req.Kcxzmcs), nil

	}

	// 异步获取成绩并更新
	go func() {
		ctx := context.Background()
		grades, err = s.getGradeFromCCNU(ctx, req.StudentID)
		if len(grades) == 0 || err != nil {
			s.l.Info("从ccnu获取成绩失败!", logger.FormatLog("ccnu", err)...)
			return
		}

		err := s.updateGrades(grades)
		if err != nil {
			s.l.Info("更新成绩失败!", logger.FormatLog("ccnu", err)...)
			return
		}
	}()

	return modelConvDomainAndFilter(grades, req.Terms, req.Kcxzmcs), nil

}

func (s *gradeService) GetGradeScore(ctx context.Context, studentId string) ([]domain.TypeOfGradeScore, error) {

	grades, err := s.gradeDAO.FindGrades(ctx, studentId, 0, 0)
	if len(grades) == 0 || err != nil {
		grades, err = s.getGradeFromCCNU(context.Background(), studentId)
		if len(grades) == 0 && err != nil {
			return nil, GET_GRADE_ERROR(err)
		}
		err = s.updateGrades(grades)
		if err != nil {
			s.l.Info("更新成绩失败!", logger.FormatLog("dao", err)...)
			return aggregateGradeScore(grades), nil
		}
		return aggregateGradeScore(grades), nil
	}

	// 异步回存
	go func() {
		grades, err := s.getGradeFromCCNU(context.Background(), studentId)
		if len(grades) == 0 && err != nil {
			s.l.Info("获取成绩失败!", logger.FormatLog("ccnu", err)...)
		}
		err = s.updateGrades(grades)
		if err != nil {
			s.l.Info("更新成绩失败!", logger.FormatLog("dao", err)...)
			return
		}
	}()

	return aggregateGradeScore(grades), nil
}

func (s *gradeService) GetUpdateScore(ctx context.Context, studentId string) ([]domain.Grade, error) {
	grades, err := s.getGradeFromCCNU(ctx, studentId)
	if len(grades) == 0 && err != nil {
		return nil, GET_GRADE_ERROR(err)
	}

	updateGrades, err := s.gradeDAO.BatchInsertOrUpdate(context.Background(), grades)
	if err != nil {
		s.l.Warn("异步回填成绩失败!", logger.FormatLog("cache", err)...)
		return nil, GET_GRADE_ERROR(err)
	}

	for _, updateGrade := range updateGrades {
		s.l.Info(
			"更新成绩成功!",
			logger.String("studentId", updateGrade.Studentid),
			logger.String("课程名称", updateGrade.Kcmc),
		)
	}

	return modelConvDomain(updateGrades), nil
}

// 包装函数
func (s *gradeService) getGradeFromCCNU(ctx context.Context, studentId string) ([]model.Grade, error) {
	// 记录开始时间
	start := time.Now()

	//尝试获取cookie
	getCookieResp, err := s.userClient.GetCookie(ctx, &userv1.GetCookieRequest{
		StudentId: studentId,
	})

	if err != nil {
		return nil, err
	}
	s.l.Warn("获取cookie花费时间:", logger.String("花费时间:", time.Since(start).String()))

	//尝试获取成绩
	items, err := GetGrade(ctx, getCookieResp.GetCookie(), 0, 0, 300)
	s.l.Warn("获取成绩花费时间:", logger.String("花费时间:", time.Since(start).String()))
	//如果获取失败成绩的话
	switch err {
	case nil:
		return items, nil

	case COOKIE_TIMEOUT:

		//尝试获取cookie
		getCookieResp, err = s.userClient.GetCookie(ctx, &userv1.GetCookieRequest{
			StudentId: studentId,
		})
		if err != nil {
			return nil, err
		}

		//尝试获取成绩
		items, err = GetGrade(ctx, getCookieResp.GetCookie(), 0, 0, 300)
		return items, err

	default:
		return nil, err
	}

}

func (s *gradeService) updateGrades(grades []model.Grade) error {
	updateGrades, err := s.gradeDAO.BatchInsertOrUpdate(context.Background(), grades)
	if err != nil {
		s.l.Warn("异步回填成绩失败!", logger.FormatLog("cache", err)...)
		return err
	}

	for _, updateGrade := range updateGrades {
		s.l.Info(
			"更新成绩成功!",
			logger.String("studentId", updateGrade.Studentid),
			logger.String("课程名称", updateGrade.Kcmc),
		)
	}
	return nil
}
