package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	userv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/user/v1"
	"github.com/asynccnu/ccnubox-be/be-grade/domain"
	"github.com/asynccnu/ccnubox-be/be-grade/pkg/logger"
	"github.com/asynccnu/ccnubox-be/be-grade/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-grade/repository/model"
)

const (
	DefaultXnmBegin = 2005 //查询范围开始为2005年
	DefaultXnmEnd   = 2100 //查询范围直到2100年
	DefaultXqmBegin = 1
	DefaultXqmEnd   = 3
)

type RankService interface {
	// grpc调用
	GetRankByTerm(ctx context.Context, req *domain.GetRankByTermReq) (*domain.GetRankByTermResp, error)
	LoadRank(ctx context.Context, req *domain.LoadRankReq)

	// cron调用
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
			return nil, err
		}
		return data, nil
	}

	ans, err := s.rankDAO.GetRankByTerm(ctx, req)
	if err != nil {
		return nil, err
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

	go s.UpdateRank(context.Background(), req.StudentId, t)
}

func (s *rankService) UpdateRank(ctx context.Context, studentId string, t *dao.Period) (*domain.GetRankByTermResp, error) {
	cookieResp, err := s.userClient.GetCookie(ctx, &userv1.GetCookieRequest{StudentId: studentId})
	if err != nil {
		// 如果是异步错误无法返回，所以输出到日志
		s.l.Warn("获取cookie出错", logger.Error(err))
		return nil, err
	}

	begin, end := ChangeToFormTime(t)

	data, err := SendReqUpdateRank(cookieResp.GetCookie(), begin, end)
	if err != nil {
		// 如果是异步错误无法返回，所以输出到日志
		s.l.Warn("向教务系统发送查询学分绩排名请求出错", logger.Error(err))
		return nil, err
	}

	err = s.rankDAO.StoreRank(ctx, convGetRankByTermFromDomainToModel(data, t, studentId))
	if err != nil {
		// 如果是异步错误无法返回，所以输出到日志
		s.l.Warn("数据库存储排名数据失败", logger.Error(err))
	}

	return data, nil
}

// 获取update为true的数据
func (s *rankService) GetRankWhichShouldUpdate(ctx context.Context, limit int, lastId int64) ([]model.Rank, error) {
	data, err := s.rankDAO.GetUpdateRank(ctx, limit, lastId)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (s *rankService) DeleteGraduateStudentRank(ctx context.Context, save int) error {
	// 例: 25年9月清21届学生的数据
	year := fmt.Sprintf("%d999999", time.Now().Year()-4-save)

	return s.rankDAO.DeleteRankByStudentId(ctx, year)
}

func (s *rankService) DeleteLessUseRank(ctx context.Context, beforeMonth int) error {
	t := time.Now().AddDate(0, beforeMonth*-1, 0)

	return s.rankDAO.DeleteRankByViewAt(ctx, t)
}

func convLoadRankRespFromModelToDomain(req *model.Rank) *domain.GetRankByTermResp {
	if req == nil {
		return nil
	}

	var j []string
	json.Unmarshal([]byte(req.Include), &j)

	return &domain.GetRankByTermResp{
		Rank:    req.Rank,
		Score:   req.Score,
		Include: j,
	}
}

func convGetRankByTermFromDomainToModel(req *domain.GetRankByTermResp, t *dao.Period, studentId string) *model.Rank {
	include, _ := json.Marshal(req.Include)
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

	// 比如25年1到9月仍属24-25年第二、三学期，此时数据不一定是最新的，所以要标记为需要自动更新
	// 为了方便管理不判断学期和月份了，以年管理
	year := int64(time.Now().Year())
	if t.XnmEnd+1 >= year {
		data.Update = true
	} else {
		data.Update = false
	}

	return data
}

// 转化成表单形式的时间格式
func ChangeToFormTime(t *dao.Period) (string, string) {
	begin := fmt.Sprintf("%d03", t.XnmBegin)
	end := fmt.Sprintf("%d03", t.XnmEnd)
	if t.XqmBegin == 2 {
		begin = fmt.Sprintf("%d12", t.XnmBegin)
	}
	if t.XqmEnd == 2 {
		end = fmt.Sprintf("%d12", t.XnmEnd)
	}
	if t.XqmBegin == 3 {
		begin = fmt.Sprintf("%d16", t.XnmBegin)
	}
	if t.XqmEnd == 3 {
		end = fmt.Sprintf("%d16", t.XnmEnd)
	}
	return begin, end
}
