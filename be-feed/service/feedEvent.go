package service

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	"github.com/asynccnu/ccnubox-be/be-feed/events/producer"
	"github.com/asynccnu/ccnubox-be/be-feed/events/topic"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/cache"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/dao"
	feedv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feed/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

// FeedEventService
type FeedEventService interface {
	GetFeedEvents(ctx context.Context, studentId string) (feedEvents []domain.FeedEventVO, fail []domain.FeedEvent, err error)
	ReadFeedEvent(ctx context.Context, id int64) error
	ClearFeedEvent(ctx context.Context, studentId string, feedId int64, status string) error
	InsertEventList(ctx context.Context, feedEvents []domain.FeedEvent) []error
	PublicFeedEvent(ctx context.Context, isAll bool, event domain.FeedEvent) error
}

// 定义错误结构体
var (
	GET_FEED_EVENT_ERROR    = errorx.FormatErrorFunc(feedv1.ErrorGetFeedEventError("获取feed失败"))
	CLEAR_FEED_EVENT_ERROR  = errorx.FormatErrorFunc(feedv1.ErrorGetFeedEventError("清除feed失败")) // 修正了文案
	PUBLIC_FEED_EVENT_ERROR = errorx.FormatErrorFunc(feedv1.ErrorPublicFeedEventError("发布feed失败"))
)

type feedEventService struct {
	feedEventDAO      dao.FeedEventDAO
	feedFailEventDAO  dao.FeedFailEventDAO
	feedEventCache    cache.FeedEventCache
	feedUserConfigDAO dao.FeedUserConfigDAO
	feedProducer      producer.Producer
	l                 logger.Logger
}

func NewFeedEventService(
	feedEventDAO dao.FeedEventDAO,
	feedEventCache cache.FeedEventCache,
	feedUserConfigDAO dao.FeedUserConfigDAO,
	feedFailEventDAO dao.FeedFailEventDAO,
	feedProducer producer.Producer,
	l logger.Logger,
) FeedEventService {
	return &feedEventService{
		feedEventCache:    feedEventCache,
		feedEventDAO:      feedEventDAO,
		feedUserConfigDAO: feedUserConfigDAO,
		feedFailEventDAO:  feedFailEventDAO,
		feedProducer:      feedProducer,
		l:                 l,
	}
}

// GetFeedEvents 根据查询条件查找 Feed 事件
func (s *feedEventService) GetFeedEvents(ctx context.Context, studentId string) (
	feedEvents []domain.FeedEventVO, fail []domain.FeedEvent, err error) {
	l := s.l.WithContext(ctx)

	events, err := s.feedEventDAO.GetFeedEventsByStudentId(ctx, studentId)
	if err != nil {
		return []domain.FeedEventVO{}, []domain.FeedEvent{}, GET_FEED_EVENT_ERROR(errorx.Errorf("service: dao get feed events failed, studentId: %s, err: %w", studentId, err))
	}
	// 转换 Model 为 VO
	feedEvents = convFeedEventFromModelToDomainVO(events)

	// 取出失败消息 (此处错误非致命，记录日志并跳过)
	failEvents, err := s.feedFailEventDAO.GetFeedFailEventsByStudentId(ctx, studentId)
	if err != nil {
		l.Warn("service: get feed fail events ignored", logger.String("studentId", studentId), logger.Error(err))
		return feedEvents, []domain.FeedEvent{}, nil
	}

	err = s.feedFailEventDAO.DelFeedFailEventsByStudentId(ctx, studentId)
	if err != nil {
		l.Warn("service: delete feed fail events ignored", logger.String("studentId", studentId), logger.Error(err))
		return feedEvents, []domain.FeedEvent{}, nil
	}

	if len(failEvents) > 0 {
		fail = convFeedFailEventFromModelToDomain(failEvents)
	}

	return feedEvents, fail, nil
}

func (s *feedEventService) ReadFeedEvent(ctx context.Context, id int64) error {
	feedEvent, err := s.feedEventDAO.GetFeedEventById(ctx, id)
	if err != nil {
		return errorx.Errorf("service: get feed event by id failed, id: %d, err: %w", id, err)
	}

	// 更新读取状态
	feedEvent.Read = true
	err = s.feedEventDAO.SaveFeedEvent(ctx, *feedEvent)
	if err != nil {
		return errorx.Errorf("service: save feed event read status failed, id: %d, err: %w", id, err)
	}
	return nil
}

// ClearFeedEvent 清除指定用户的所有 Feed 事件
func (s *feedEventService) ClearFeedEvent(ctx context.Context, studentId string, feedEventId int64, status string) error {
	l := s.l.WithContext(ctx)
	if feedEventId == 0 && status == "" {
		l.Info("service: clear feed event skip, missing params", logger.String("studentId", studentId))
		return nil
	}

	err := s.feedEventDAO.RemoveFeedEvent(ctx, studentId, feedEventId, status)
	if err != nil {
		return CLEAR_FEED_EVENT_ERROR(errorx.Errorf("service: dao remove feed event failed, studentId: %s, feedId: %d, status: %s, err: %w", studentId, feedEventId, status, err))
	}

	return nil
}

func (s *feedEventService) InsertEventList(ctx context.Context, feedEvents []domain.FeedEvent) []error {
	var errs []error
	l := s.l.WithContext(ctx)

	_, err := s.feedEventDAO.InsertFeedEventList(ctx, convFeedEventsFromDomainToModel(feedEvents))
	if err != nil {
		l.Error("service: batch insert feedEvent failed, trying fallback to individual insert", logger.Error(err))
		for i := range feedEvents {
			_, err = s.feedEventDAO.InsertFeedEvent(ctx, convFeedEventFromDomainToModel(&feedEvents[i]))
			if err != nil {
				wrappedErr := errorx.Errorf("service: individual insert failed, studentId: %s, err: %w", feedEvents[i].StudentId, err)
				l.Error("插入feedEvent失败", logger.Error(wrappedErr))
				errs = append(errs, wrappedErr)
			}
		}
	}
	return errs
}

func (s *feedEventService) PublicFeedEvent(ctx context.Context, isAll bool, event domain.FeedEvent) error {
	l := s.l.WithContext(ctx)

	if isAll {
		const batchSize = 50
		var lastId int64 = 0

		for {
			studentIds, newLastId, err := s.feedUserConfigDAO.GetStudentIdsByCursor(ctx, lastId, batchSize)
			if err != nil {
				return PUBLIC_FEED_EVENT_ERROR(errorx.Errorf("service: get student ids by cursor failed, lastId: %d, err: %w", lastId, err))
			}

			if len(studentIds) == 0 {
				return nil
			}

			for i := range studentIds {
				event.StudentId = studentIds[i]
				err := s.feedProducer.SendMessage(topic.FeedEvent, event)
				if err != nil {
					// 批量推送中的单个失败记录日志，不中断循环
					l.Error("service: batch send message failed",
						logger.Error(err),
						logger.String("studentId", studentIds[i]))
				}
			}

			lastId = newLastId
		}
	}

	err := s.feedProducer.SendMessage(topic.FeedEvent, event)
	if err != nil {
		return PUBLIC_FEED_EVENT_ERROR(errorx.Errorf("service: send single message failed, studentId: %s, err: %w", event.StudentId, err))
	}
	return nil
}
