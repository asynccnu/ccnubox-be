package service

import (
	"fmt"
	"strings"

	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/cache"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/model"
	feedv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feed/v1"
)

func convFeedEventsFromModelToDomain(feedEvents []model.FeedEvent) []domain.FeedEvent {
	result := make([]domain.FeedEvent, len(feedEvents)) // 直接预分配
	for i := range feedEvents {
		result[i] = domain.FeedEvent{ // 通过索引直接赋值
			ID:           feedEvents[i].ID,
			StudentId:    feedEvents[i].StudentId,
			Type:         feedEvents[i].Type,
			Title:        feedEvents[i].Title,
			Content:      feedEvents[i].Content,
			ExtendFields: feedEvents[i].ExtendFields,
			CreatedAt:    feedEvents[i].CreatedAt,
		}
	}
	return result
}

func convFeedEventFromDomainToModel(feedEvent *domain.FeedEvent) *model.FeedEvent {
	return &model.FeedEvent{ // 通过索引直接赋值
		StudentId:    feedEvent.StudentId,
		Type:         feedEvent.Type,
		Title:        feedEvent.Title,
		Content:      feedEvent.Content,
		ExtendFields: feedEvent.ExtendFields,
	}
}

func convFeedEventsFromDomainToModel(feedEvents []domain.FeedEvent) []model.FeedEvent {
	result := make([]model.FeedEvent, len(feedEvents)) // 直接预分配
	for i := range feedEvents {
		result[i] = model.FeedEvent{ // 通过索引直接赋值
			StudentId:    feedEvents[i].StudentId,
			Type:         feedEvents[i].Type,
			Title:        feedEvents[i].Title,
			Content:      feedEvents[i].Content,
			ExtendFields: feedEvents[i].ExtendFields,
		}
	}
	return result
}

func convFeedEventFromModelToDomainVO(feedEvents []model.FeedEvent) []domain.FeedEventVO {
	result := make([]domain.FeedEventVO, len(feedEvents)) // 直接预分配
	for i := range feedEvents {
		result[i] = domain.FeedEventVO{ // 通过索引直接赋值
			ID:           feedEvents[i].ID,
			StudentId:    feedEvents[i].StudentId,
			Type:         feedEvents[i].Type,
			Title:        feedEvents[i].Title,
			Content:      feedEvents[i].Content,
			ExtendFields: feedEvents[i].ExtendFields,
			CreatedAt:    feedEvents[i].CreatedAt,
			Read:         feedEvents[i].Read,
			Route:        joinRoutes(feedEvents[i]),
		}
	}
	return result
}
func convFeedFailEventFromModelToDomain(feedEvents []model.FeedFailEvent) []domain.FeedEvent {
	result := make([]domain.FeedEvent, len(feedEvents)) // 直接预分配
	for i := range feedEvents {
		result[i] = domain.FeedEvent{ // 通过索引直接赋值
			ID:           feedEvents[i].ID,
			StudentId:    feedEvents[i].StudentId,
			Type:         feedEvents[i].Type,
			Title:        feedEvents[i].Title,
			Content:      feedEvents[i].Content,
			ExtendFields: feedEvents[i].ExtendFields,
			CreatedAt:    feedEvents[i].CreatedAt,
		}
	}
	return result
}
func convFeedFailEventFromDomainToModel(feedEvents []domain.FeedEvent) []model.FeedFailEvent {
	result := make([]model.FeedFailEvent, len(feedEvents)) // 直接预分配
	for i := range feedEvents {
		result[i] = model.FeedFailEvent{
			StudentId:    feedEvents[i].StudentId,
			Type:         feedEvents[i].Type,
			Title:        feedEvents[i].Title,
			Content:      feedEvents[i].Content,
			ExtendFields: feedEvents[i].ExtendFields,
		}
	}
	return result
}

func convMuxiMessageFromCacheToDomain(feeds []cache.MuxiOfficialMSG) []domain.MuxiOfficialMSG {
	//类型转换
	result := make([]domain.MuxiOfficialMSG, len(feeds))
	for i := range feeds {
		result[i] = domain.MuxiOfficialMSG{
			Id:           feeds[i].MuxiMSGId,
			Title:        feeds[i].Title,
			Content:      feeds[i].Content,
			ExtendFields: domain.ExtendFields(feeds[i].ExtendFields),
			//PublicTime:   feeds[i].PublicTime,
		}
	}
	return result
}

func joinRoutes(feedEvent model.FeedEvent) string {
	fn, ok := routeRules[feedEvent.Type]
	if !ok {
		return ""
	}
	return fn(feedEvent)
}

var routeRules = map[string]func(vo model.FeedEvent) string{
	strings.ToLower(feedv1.FeedEventType_GRADE.String()): func(vo model.FeedEvent) string {
		return "/scoreInquiry"
	},
	strings.ToLower(feedv1.FeedEventType_HOLIDAY.String()): func(vo model.FeedEvent) string {
		return "/calendar"
	},
	strings.ToLower(feedv1.FeedEventType_MUXI.String()): func(vo model.FeedEvent) string {
		return ""
	},
	strings.ToLower(feedv1.FeedEventType_ENERGY.String()): func(vo model.FeedEvent) string {
		return "/electricity"
	},
	strings.ToLower(feedv1.FeedEventType_FEEDBACK.String()): func(vo model.FeedEvent) string {
		return fmt.Sprintf("/feedback/%s", vo.ExtendFields["record_id"])
	},
}
