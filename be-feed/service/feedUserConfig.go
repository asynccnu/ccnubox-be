package service

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/cache"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/model"
	feedv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feed/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/tool"
	"golang.org/x/exp/slices"
)

type FeedUserConfigService interface {
	ChangeAllowList(ctx context.Context, req domain.AllowList) error
	FindOrCreateAllowList(ctx context.Context, studentId string) (domain.AllowList, error)
	IsLibraryEnabled(ctx context.Context, studentID string) (bool, error)
	ListLibraryPreferenceChanges(ctx context.Context, afterRevision int64, limit int) ([]domain.LibraryPreferenceChange, int64, error)
	ListLibraryReminderUsers(ctx context.Context, afterID, snapshotRevision int64, limit int) ([]domain.LibraryReminderUser, int64, int64, error)
	SaveFeedToken(ctx context.Context, studentId string, token string) error
	GetFeedTokens(ctx context.Context, studentId string) (tokens []string, err error)
	RemoveFeedToken(ctx context.Context, studentId string, token string) error
}

// 使用封装好的 map 获取对应位的位置信息
var configMap = map[string]int{
	"muxi":     model.MuxiPos,
	"grade":    model.GradePos,
	"energy":   model.EnergyPos,
	"holiday":  model.HolidayPos,
	"feedback": model.FeedBackPos,
	"library":  model.LibraryPos,
}

type feedUserConfigService struct {
	feedEventCache    cache.FeedEventCache
	userFeedConfigDAO dao.FeedUserConfigDAO
	feedTokenDAO      dao.FeedTokenDAO
}

func NewFeedUserConfigService(
	feedEventCache cache.FeedEventCache,
	feedAllowListEventDAO dao.FeedUserConfigDAO,
	tokenFeedDAO dao.FeedTokenDAO,
) FeedUserConfigService {
	return &feedUserConfigService{
		feedEventCache:    feedEventCache,
		userFeedConfigDAO: feedAllowListEventDAO,
		feedTokenDAO:      tokenFeedDAO,
	}
}

// 定义错误结构体
var (
	FIND_CONFIG_OR_TOKEN_ERROR   = errorx.FormatErrorFunc(feedv1.ErrorFindConfigOrTokenError("获取推送配置失败"))
	CHANGE_CONFIG_OR_TOKEN_ERROR = errorx.FormatErrorFunc(feedv1.ErrorChangeConfigOrTokenError("更改推送配置失败"))
	REMOVE_CONFIG_OR_TOKEN_ERROR = errorx.FormatErrorFunc(feedv1.ErrorRemoveConfigOrTokenError("删除推送配置失败"))
)

// ChangeAllowList 修改允许列表
func (s *feedUserConfigService) ChangeAllowList(ctx context.Context, req domain.AllowList) error {
	if !tool.IsValidStudentID(req.StudentId) {
		return CHANGE_CONFIG_OR_TOKEN_ERROR(errorx.New("service: invalid student id"))
	}
	bits := map[int]bool{
		model.GradePos:    req.Grade,
		model.MuxiPos:     req.Muxi,
		model.HolidayPos:  req.Holiday,
		model.EnergyPos:   req.Energy,
		model.FeedBackPos: req.FeedBack,
	}
	_, err := s.userFeedConfigDAO.ChangeConfigBits(ctx, req.StudentId, bits, req.Library)
	if err != nil {
		return CHANGE_CONFIG_OR_TOKEN_ERROR(errorx.Errorf("service: change user feed config failed, sid: %s, err: %w", req.StudentId, err))
	}
	return nil
}

func (s *feedUserConfigService) FindOrCreateAllowList(ctx context.Context, studentId string) (domain.AllowList, error) {
	if !tool.IsValidStudentID(studentId) {
		return domain.AllowList{}, FIND_CONFIG_OR_TOKEN_ERROR(errorx.New("service: invalid student id"))
	}
	list, err := s.userFeedConfigDAO.FindOrCreateUserFeedConfig(ctx, studentId)
	if err != nil {
		return domain.AllowList{}, FIND_CONFIG_OR_TOKEN_ERROR(errorx.Errorf("service: find or create allow list failed, sid: %s, err: %w", studentId, err))
	}
	library := s.userFeedConfigDAO.GetConfigBit(list.PushConfig, model.LibraryPos)
	return domain.AllowList{
		StudentId: list.StudentId,
		Grade:     s.userFeedConfigDAO.GetConfigBit(list.PushConfig, model.GradePos),
		Muxi:      s.userFeedConfigDAO.GetConfigBit(list.PushConfig, model.MuxiPos),
		Holiday:   s.userFeedConfigDAO.GetConfigBit(list.PushConfig, model.HolidayPos),
		Energy:    s.userFeedConfigDAO.GetConfigBit(list.PushConfig, model.EnergyPos),
		FeedBack:  s.userFeedConfigDAO.GetConfigBit(list.PushConfig, model.FeedBackPos),
		Library:   &library,
	}, nil
}

func (s *feedUserConfigService) IsLibraryEnabled(ctx context.Context, studentID string) (bool, error) {
	return s.userFeedConfigDAO.IsLibraryEnabled(ctx, studentID)
}

func preferencePageLimit(limit int) int {
	if limit <= 0 {
		return 100
	}
	if limit > 1000 {
		return 1000
	}
	return limit
}

func (s *feedUserConfigService) ListLibraryPreferenceChanges(ctx context.Context, afterRevision int64, limit int) ([]domain.LibraryPreferenceChange, int64, error) {
	changes, err := s.userFeedConfigDAO.ListLibraryPreferenceChanges(ctx, afterRevision, preferencePageLimit(limit))
	if err != nil {
		return nil, afterRevision, FIND_CONFIG_OR_TOKEN_ERROR(err)
	}
	result := make([]domain.LibraryPreferenceChange, len(changes))
	next := afterRevision
	for i := range changes {
		result[i] = domain.LibraryPreferenceChange{
			Revision:  changes[i].Revision,
			StudentID: changes[i].StudentId,
			Enabled:   changes[i].LibraryEnabled,
			ChangedAt: changes[i].CreatedAt,
		}
		next = changes[i].Revision
	}
	return result, next, nil
}

func (s *feedUserConfigService) ListLibraryReminderUsers(ctx context.Context, afterID, snapshotRevision int64, limit int) ([]domain.LibraryReminderUser, int64, int64, error) {
	if snapshotRevision == 0 {
		var err error
		snapshotRevision, err = s.userFeedConfigDAO.LatestLibraryPreferenceRevision(ctx)
		if err != nil {
			return nil, afterID, 0, FIND_CONFIG_OR_TOKEN_ERROR(err)
		}
	}
	configs, err := s.userFeedConfigDAO.ListLibraryReminderUsers(ctx, afterID, snapshotRevision, preferencePageLimit(limit))
	if err != nil {
		return nil, afterID, snapshotRevision, FIND_CONFIG_OR_TOKEN_ERROR(err)
	}
	result := make([]domain.LibraryReminderUser, len(configs))
	next := afterID
	for i := range configs {
		result[i] = domain.LibraryReminderUser{
			ID:        configs[i].ID,
			StudentID: configs[i].StudentId,
			Revision:  configs[i].LibraryRevision,
		}
		next = configs[i].ID
	}
	return result, next, snapshotRevision, nil
}

func (s *feedUserConfigService) SaveFeedToken(ctx context.Context, studentId string, token string) error {
	tokens, err := s.feedTokenDAO.GetTokens(ctx, studentId)
	if err != nil {
		return FIND_CONFIG_OR_TOKEN_ERROR(errorx.Errorf("service: get current tokens failed before saving, sid: %s, err: %w", studentId, err))
	}

	if token != "" && !slices.Contains(tokens, token) {
		err = s.feedTokenDAO.AddToken(ctx, studentId, token)
		if err != nil {
			return CHANGE_CONFIG_OR_TOKEN_ERROR(errorx.Errorf("service: add new token failed, sid: %s, err: %w", studentId, err))
		}
	}
	return nil
}

func (s *feedUserConfigService) GetFeedTokens(ctx context.Context, studentId string) (tokens []string, err error) {
	tokens, err = s.feedTokenDAO.GetTokens(ctx, studentId)
	if err != nil {
		return []string{}, FIND_CONFIG_OR_TOKEN_ERROR(errorx.Errorf("service: get feed tokens failed, sid: %s, err: %w", studentId, err))
	}
	return tokens, nil
}

func (s *feedUserConfigService) RemoveFeedToken(ctx context.Context, studentId string, token string) error {
	err := s.feedTokenDAO.RemoveToken(ctx, studentId, token)
	if err != nil {
		return REMOVE_CONFIG_OR_TOKEN_ERROR(errorx.Errorf("service: remove token failed, sid: %s, err: %w", studentId, err))
	}
	return nil
}
