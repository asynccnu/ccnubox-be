package grpc

import (
	feedv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/feed/v1"
	"github.com/asynccnu/ccnubox-be/be-feed/domain"
)

// 好长的函数名称
func convAllowListFromGRPCToDomain(list *feedv1.AllowList) domain.AllowList {
	return domain.AllowList{
		StudentId: list.StudentId,
		Grade:     list.Grade,
		Muxi:      list.Muxi,
		Holiday:   list.Holiday,
		Energy:    list.Energy,
	}
}

func convAllowListFromDomainToGRPC(list *domain.AllowList) *feedv1.AllowList {
	return &feedv1.AllowList{
		StudentId: list.StudentId,
		Grade:     list.Grade,
		Muxi:      list.Muxi,
		Holiday:   list.Holiday,
		Energy:    list.Energy,
	}
}

func convFeedEventsVOFromDomainToGRPC(feedEvents []domain.FeedEventVO) []*feedv1.FeedEventVO {
	result := make([]*feedv1.FeedEventVO, len(feedEvents))
	for i := range feedEvents {
		result[i] = &feedv1.FeedEventVO{
			Id:           feedEvents[i].ID,
			Type:         feedEvents[i].Type,
			Title:        feedEvents[i].Title,
			Content:      feedEvents[i].Content,
			ExtendFields: feedEvents[i].ExtendFields,
			CreatedAt:    feedEvents[i].CreatedAt,
			Read:         feedEvents[i].Read,
		}
	}
	return result
}

func convFeedEventsFromGRPCToDomain(feedEvents []*feedv1.FeedEvent) []domain.FeedEvent {
	result := make([]domain.FeedEvent, 0, len(feedEvents))
	for i := range feedEvents {
		result[i] = domain.FeedEvent{
			ID:           feedEvents[i].GetId(),
			Type:         feedEvents[i].GetType(),
			Title:        feedEvents[i].GetTitle(),
			Content:      feedEvents[i].GetContent(),
			ExtendFields: feedEvents[i].GetExtendFields(),
			CreatedAt:    feedEvents[i].GetCreatedAt(),
		}

	}
	return result
}

func convMuxiMSGFromGRPCTODomain(msg *feedv1.MuxiOfficialMSG) *domain.MuxiOfficialMSG {
	return &domain.MuxiOfficialMSG{
		Title:        msg.Title,
		Content:      msg.Content,
		ExtendFields: msg.ExtendFields,
		PublicTime:   msg.PublicTime,
		Id:           msg.Id,
	}
}
func convMuxiMSGFromDomainTOGRPC(msg *domain.MuxiOfficialMSG) *feedv1.MuxiOfficialMSG {
	return &feedv1.MuxiOfficialMSG{
		Title:        msg.Title,
		Content:      msg.Content,
		ExtendFields: msg.ExtendFields,
		PublicTime:   msg.PublicTime,
		Id:           msg.Id,
	}
}
