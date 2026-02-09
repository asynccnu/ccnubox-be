package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/be-grade/domain"
	"github.com/asynccnu/ccnubox-be/be-grade/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-grade/repository/model"
	userv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/user/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

const (
	DefaultXnmBegin = 2005
	DefaultXnmEnd   = 2100
	DefaultXqmBegin = 1
	DefaultXqmEnd   = 3
)

type RankService interface {
	GetRankByTerm(ctx context.Context, req *domain.GetRankByTermReq) (*domain.GetRankByTermResp, error)
	LoadRank(ctx context.Context, req *domain.LoadRankReq)
	GetRankWhichShouldUpdate(ctx context.Context, limit int, lastId int64) ([]model.Rank, error)
	UpdateRank(ctx context.Context, studentId string, t *dao.Period) (*domain.GetRankByTermResp, error)
	DeleteGraduateStudentRank(ctx context.Context, save int) error
	DeleteLessUseRank(ctx context.Context, beforeMonth int) error
}

type rankService struct {
	userClient userv1.UserServiceClient
	rankDAO    dao.RankDAO
	l          logger.Logger
}

func NewRankService(rankDAO dao.RankDAO, l logger.Logger, userClient userv1.UserServiceClient) RankService {
	return &rankService{rankDAO: rankDAO, l: l, userClient: userClient}
}

func (s *rankService) GetRankByTerm(ctx context.Context, req *domain.GetRankByTermReq) (*domain.GetRankByTermResp, error) {
	t := &dao.Period{
		XnmBegin: req.XnmBegin,
		XnmEnd:   req.XnmEnd,
		XqmBegin: req.XqmBegin,
		XqmEnd:   req.XqmEnd,
	}

	// 强制刷新或者不存在对应数据就阻塞查询
	if req.Refresh || !s.rankDAO.RankExist(ctx, req.StudentId, t) {
		data, err := s.UpdateRank(ctx, req.StudentId, t)
		if err != nil {
			return nil, errorx.Errorf("service: get rank by term failed during update, sid: %s, err: %w", req.StudentId, err)
		}
		return data, nil
	}

	ans, err := s.rankDAO.GetRankByTerm(ctx, req)
	if err != nil {
		return nil, errorx.Errorf("service: get rank from dao failed, sid: %s, err: %w", req.StudentId, err)
	}

	return convLoadRankRespFromModelToDomain(ans), nil
}

func (s *rankService) LoadRank(ctx context.Context, req *domain.LoadRankReq) {
	t := &dao.Period{
		XnmBegin: DefaultXnmBegin,
		XqmBegin: DefaultXqmBegin,
		XqmEnd:   DefaultXqmEnd,
		XnmEnd:   DefaultXnmEnd,
	}
	if s.rankDAO.RankExist(ctx, req.StudentId, t) {
		return
	}

	// 异步更新排名，由于无法给用户返回 error，需记录 Warn 级别日志
	go func() {
		_, err := s.UpdateRank(context.Background(), req.StudentId, t)
		if err != nil {
			s.l.Warn("service: async load rank failed", logger.String("sid", req.StudentId), logger.Error(err))
		}
	}()
}

func (s *rankService) UpdateRank(ctx context.Context, studentId string, t *dao.Period) (*domain.GetRankByTermResp, error) {
	l := s.l.WithContext(ctx)
	cookieResp, err := s.userClient.GetCookie(ctx, &userv1.GetCookieRequest{StudentId: studentId})
	if err != nil {
		return nil, errorx.Errorf("service: rpc get cookie failed, sid: %s, err: %w", studentId, err)
	}

	begin, end := ChangeToFormTime(t)

	data, err := SendReqUpdateRank(cookieResp.GetCookie(), begin, end)
	if err != nil {
		return nil, errorx.Errorf("service: fetch rank from school system failed, sid: %s, period: %s-%s, err: %w", studentId, begin, end, err)
	}

	modelData, err := convGetRankByTermFromDomainToModel(data, t, studentId)
	if err != nil {
		return nil, errorx.Errorf("service: convert domain to model failed, sid: %s, err: %w", studentId, err)
	}

	err = s.rankDAO.StoreRank(ctx, modelData)
	if err != nil {
		// 这里不返回 error 以保证数据即使存库失败也可能返回给前端展示
		l.Error("service: store rank to db failed", logger.String("sid", studentId), logger.Error(err))
	}

	return data, nil
}

func (s *rankService) GetRankWhichShouldUpdate(ctx context.Context, limit int, lastId int64) ([]model.Rank, error) {
	data, err := s.rankDAO.GetUpdateRank(ctx, limit, lastId)
	if err != nil {
		return nil, errorx.Errorf("service: dao get update rank list failed, lastId: %d, err: %w", lastId, err)
	}
	return data, nil
}

func (s *rankService) DeleteGraduateStudentRank(ctx context.Context, save int) error {
	// 例: 25年9月清21届学生的数据
	year := fmt.Sprintf("%d999999", time.Now().Year()-4-save)
	err := s.rankDAO.DeleteRankByStudentId(ctx, year)
	if err != nil {
		return errorx.Errorf("service: delete graduate student rank failed, year_prefix: %s, err: %w", year, err)
	}
	return nil
}

func (s *rankService) DeleteLessUseRank(ctx context.Context, beforeMonth int) error {
	t := time.Now().AddDate(0, beforeMonth*-1, 0)
	err := s.rankDAO.DeleteRankByViewAt(ctx, t)
	if err != nil {
		return errorx.Errorf("service: delete less use rank failed, before_time: %v, err: %w", t, err)
	}
	return nil
}

// 辅助转换函数增加了错误检查
func convLoadRankRespFromModelToDomain(req *model.Rank) *domain.GetRankByTermResp {
	if req == nil {
		return nil
	}

	var j []string
	_ = json.Unmarshal([]byte(req.Include), &j)

	return &domain.GetRankByTermResp{
		Rank:    req.Rank,
		Score:   req.Score,
		Include: j,
	}
}

func convGetRankByTermFromDomainToModel(req *domain.GetRankByTermResp, t *dao.Period, studentId string) (*model.Rank, error) {
	include, err := json.Marshal(req.Include)
	if err != nil {
		return nil, errorx.Errorf("json marshal include failed: %w", err)
	}

	data := &model.Rank{
		StudentId: studentId,
		Rank:      req.Rank,
		Score:     req.Score,
		Include:   string(include),
		XnmBegin:  t.XnmBegin,
		XqmBegin:  t.XqmBegin,
		XnmEnd:    t.XnmEnd,
		XqmEnd:    t.XqmEnd,
		ViewAt:    time.Now(),
	}

	// 自动更新标记逻辑
	year := int64(time.Now().Year())
	if t.XnmEnd+1 >= year {
		data.Update = true
	} else {
		data.Update = false
	}

	return data, nil
}

func ChangeToFormTime(t *dao.Period) (string, string) {
	format := func(xnm int64, xqm int64) string {
		suffix := "03" // 默认第一学期
		switch xqm {
		case 2:
			suffix = "12"
		case 3:
			suffix = "16"
		}
		return fmt.Sprintf("%d%s", xnm, suffix)
	}
	return format(t.XnmBegin, t.XqmBegin), format(t.XnmEnd, t.XqmEnd)
}
