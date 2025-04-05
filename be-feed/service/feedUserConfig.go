package service

import (
	"context"
	feedv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/feed/v1"
	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	"github.com/asynccnu/ccnubox-be/be-feed/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/cache"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/model"
	"golang.org/x/exp/slices"
	"reflect"
)

type FeedUserConfigService interface {
	ChangeAllowList(ctx context.Context, req domain.AllowList) error
	GetFeedAllowList(ctx context.Context, studentId string) (domain.AllowList, error)
	SaveFeedToken(ctx context.Context, studentId string, token string) error
	GetFeedTokens(ctx context.Context, studentId string) (tokens []string, err error)
	RemoveFeedToken(ctx context.Context, studentId string, token string) error
}

// 使用封装好的 map 获取对应位的位置信息
var configMap = map[string]int{
	"muxi":    model.MuxiPos,
	"grade":   model.GradePos,
	"energy":  model.EnergyPos,
	"holiday": model.HolidayPos,
}

type feedUserConfigService struct {
	feedEventDAO      dao.FeedEventDAO
	feedEventCache    cache.FeedEventCache
	userFeedConfigDAO dao.UserFeedConfigDAO
	feedTokenDAO      dao.UserFeedTokenDAO
}

func NewFeedUserConfigService(
	feedEventDAO dao.FeedEventDAO,
	feedEventCache cache.FeedEventCache,
	feedAllowListEventDAO dao.UserFeedConfigDAO,
	tokenFeedDAO dao.UserFeedTokenDAO,
) FeedUserConfigService {
	return &feedUserConfigService{
		feedEventCache:    feedEventCache,
		feedEventDAO:      feedEventDAO,
		userFeedConfigDAO: feedAllowListEventDAO,
		feedTokenDAO:      tokenFeedDAO,
	}
}

// 定义错误结构体
var (
	FIND_CONFIG_OR_TOKEN_ERROR = func(err error) error {
		return errorx.New(feedv1.ErrorFindConfigOrTokenError("获取推送配置失败"), "dao", err)
	}

	CHANGE_CONFIG_OR_TOKEN_ERROR = func(err error) error {
		return errorx.New(feedv1.ErrorChangeConfigOrTokenError("更改推送配置失败"), "dao", err)
	}

	REMOVE_CONFIG_OR_TOKEN_ERROR = func(err error) error {
		return errorx.New(feedv1.ErrorRemoveConfigOrTokenError("删除推送配置失败"), "dao", err)
	}
)

// ChangeAllowList 修改允许列表
func (s *feedUserConfigService) ChangeAllowList(ctx context.Context, req domain.AllowList) error {
	list, err := s.userFeedConfigDAO.FindOrCreateUserFeedConfig(ctx, req.StudentId)
	if err != nil {
		return FIND_CONFIG_OR_TOKEN_ERROR(err)
	}

	// 定义映射关系：字段名 -> 对应的 bit 位
	bitMap := map[string]int{
		"Grade":     model.GradePos,
		"Muxi":      model.MuxiPos,
		"Holiday":   model.HolidayPos,
		"EnergyPos": model.EnergyPos,
	}

	// 反射获取字段值，并修改 pushConfig
	val := reflect.ValueOf(req)
	for field, bitPos := range bitMap {
		fieldValue := val.FieldByName(field)
		if fieldValue.IsValid() && fieldValue.Kind() == reflect.Bool {
			if fieldValue.Bool() {
				s.userFeedConfigDAO.SetConfigBit(&list.PushConfig, bitPos)
			} else {
				s.userFeedConfigDAO.ClearConfigBit(&list.PushConfig, bitPos)
			}
		}
	}

	//更新配置
	err = s.userFeedConfigDAO.SaveUserFeedConfig(ctx, list)
	if err != nil {
		return CHANGE_CONFIG_OR_TOKEN_ERROR(err)
	}

	// 调用 DAO 层的修改方法，更新允许列表
	return nil
}

func (s *feedUserConfigService) GetFeedAllowList(ctx context.Context, studentId string) (domain.AllowList, error) {
	list, err := s.userFeedConfigDAO.FindOrCreateUserFeedConfig(ctx, studentId)
	if err != nil {
		return domain.AllowList{}, FIND_CONFIG_OR_TOKEN_ERROR(err)
	}
	return domain.AllowList{
		StudentId: list.StudentId,
		Grade:     s.userFeedConfigDAO.GetConfigBit(list.PushConfig, model.GradePos),
		Muxi:      s.userFeedConfigDAO.GetConfigBit(list.PushConfig, model.MuxiPos),
		Holiday:   s.userFeedConfigDAO.GetConfigBit(list.PushConfig, model.HolidayPos),
		Energy:    s.userFeedConfigDAO.GetConfigBit(list.PushConfig, model.EnergyPos),
	}, nil
}

func (s *feedUserConfigService) SaveFeedToken(ctx context.Context, studentId string, token string) error {
	tokens, err := s.feedTokenDAO.GetTokens(ctx, studentId)
	if err != nil {
		return FIND_CONFIG_OR_TOKEN_ERROR(err)
	}

	if token != "" && !slices.Contains(tokens, token) {
		err = s.feedTokenDAO.AddToken(ctx, studentId, token)
		if err != nil {
			return CHANGE_CONFIG_OR_TOKEN_ERROR(err)
		}
		return nil
	} else {
		return nil
	}
}

func (s *feedUserConfigService) GetFeedTokens(ctx context.Context, studentId string) (tokens []string, err error) {
	tokens, err = s.feedTokenDAO.GetTokens(ctx, studentId)
	if err != nil {
		return []string{}, FIND_CONFIG_OR_TOKEN_ERROR(err)
	}
	return tokens, nil
}

func (s *feedUserConfigService) RemoveFeedToken(ctx context.Context, studentId string, token string) error {
	err := s.feedTokenDAO.RemoveToken(ctx, studentId, token)
	if err != nil {
		return REMOVE_CONFIG_OR_TOKEN_ERROR(err)
	}
	return nil
}
