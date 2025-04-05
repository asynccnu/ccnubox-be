package service

import (
	"context"
	"errors"
	"fmt"
	feedv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/feed/v1"
	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	"github.com/asynccnu/ccnubox-be/be-feed/events/producer"
	"github.com/asynccnu/ccnubox-be/be-feed/events/topic"
	"github.com/asynccnu/ccnubox-be/be-feed/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/be-feed/pkg/logger"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/cache"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/dao"
)

// FeedEventService
type FeedEventService interface {
	GetFeedEvents(ctx context.Context, studentId string) (
		feedEvents []domain.FeedEventVO, fail []domain.FeedEvent, err error)
	ReadFeedEvent(ctx context.Context, id int64) error
	ClearFeedEvent(ctx context.Context, studentId string, feedId int64, status string) error
	InsertEventList(ctx context.Context, feedEvents []domain.FeedEvent) []error
	PublicFeedEvent(ctx context.Context, isAll bool, event domain.FeedEvent) error
}

// 定义错误结构体
var (
	GET_FEED_EVENT_ERROR = func(err error) error {
		return errorx.New(feedv1.ErrorGetFeedEventError("获取feed失败"), "dao", err)
	}

	CLEAR_FEED_EVENT_ERROR = func(err error) error {
		return errorx.New(feedv1.ErrorClearFeedEventError("删除feed失败"), "dao", err)
	}
	PUBLIC_FEED_EVENT_ERROR = func(err error) error {
		return errorx.New(feedv1.ErrorPublicFeedEventError("发布feed失败"), "dao", err)
	}
)

type feedEventService struct {
	feedEventDAO     dao.FeedEventDAO
	feedFailEventDAO dao.FeedFailEventDAO

	feedEventCache    cache.FeedEventCache
	userFeedConfigDAO dao.UserFeedConfigDAO
	feedProducer      producer.Producer
	l                 logger.Logger
}

func NewFeedEventService(
	feedEventDAO dao.FeedEventDAO,
	feedEventCache cache.FeedEventCache,
	userFeedConfigDAO dao.UserFeedConfigDAO,
	feedFailEventDAO dao.FeedFailEventDAO,
	feedProducer producer.Producer,
	l logger.Logger,
) FeedEventService {
	return &feedEventService{
		feedEventCache:    feedEventCache,
		feedEventDAO:      feedEventDAO,
		userFeedConfigDAO: userFeedConfigDAO,
		feedFailEventDAO:  feedFailEventDAO,
		feedProducer:      feedProducer,
		l:                 l,
	}
}

// FindPushFeedEvents 根据查询条件查找 Feed 事件
func (s *feedEventService) GetFeedEvents(ctx context.Context, studentId string) (
	feedEvents []domain.FeedEventVO, fail []domain.FeedEvent, err error) {
	events, err := s.feedEventDAO.GetFeedEventsByStudentId(ctx, studentId)
	if err != nil {
		return []domain.FeedEventVO{}, []domain.FeedEvent{}, GET_FEED_EVENT_ERROR(err)
	}

	for i := 0; i < len(events); i++ {
		feedEvents = convFeedEventFromModelToDomainVO(events)
	}

	//取出失败消息
	failEvents, err := s.feedFailEventDAO.GetFeedFailEventsByStudentId(ctx, studentId)
	if err != nil {
		return feedEvents, []domain.FeedEvent{}, nil
	}

	err = s.feedFailEventDAO.DelFeedFailEventsByStudentId(ctx, studentId)
	if err != nil {
		return feedEvents, []domain.FeedEvent{}, nil
	}

	//如果有失败数据则更新
	if len(failEvents) > 0 {
		fail = convFeedFailEventFromModelToDomain(failEvents)
	}

	// 调用 DAO 层的查找方法，返回数据
	return feedEvents, fail, nil
}

func (s *feedEventService) ReadFeedEvent(ctx context.Context, id int64) error {
	feedEvent, err := s.feedEventDAO.GetFeedEventById(ctx, id)
	if err != nil {
		return err
	}
	//更新读取状态
	feedEvent.Read = true
	err = s.feedEventDAO.SaveFeedEvent(ctx, *feedEvent)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

// ClearEvents 清除指定用户的所有 Feed 事件
func (s *feedEventService) ClearFeedEvent(ctx context.Context, studentId string, feedEventId int64, status string) error {
	// 调用 DAO 层的清除方法，删除用户的 Feed 事件
	if feedEventId == 0 && status == "" {
		s.l.Info("意外的清除规则", logger.FormatLog("params", errors.New("参数不足"))...)
		return nil
	}

	err := s.feedEventDAO.RemoveFeedEvent(ctx, studentId, feedEventId, status)
	if err != nil {
		fmt.Println(err)
		return CLEAR_FEED_EVENT_ERROR(err)
	}

	return nil
}

func (s *feedEventService) InsertEventList(ctx context.Context, feedEvents []domain.FeedEvent) []error {
	var errs []error
	// 开始事务，通过 DAO 层进行

	_, err := s.feedEventDAO.InsertFeedEventList(ctx, convFeedEventsFromDomainToModel(feedEvents))
	if err != nil {
		s.l.Error("批量插入feedEvent失败", logger.FormatLog("system", err)...)
		for i := range feedEvents {
			_, err = s.feedEventDAO.InsertFeedEvent(ctx, convFeedEventFromDomainToModel(&feedEvents[i]))
			if err != nil {
				s.l.Error("插入feedEvent失败", append(
					logger.FormatLog("system", err),
					logger.String("feedData", fmt.Sprintf("%v", feedEvents[i])),
				)...,
				)
				errs = append(errs, err)
			}
		}
	}
	return errs
}

func (s *feedEventService) PublicFeedEvent(ctx context.Context, isAll bool, event domain.FeedEvent) error {

	if isAll {

		const batchSize = 50 // 每批次处理的用户数(为什么一次只推送50条呢?主要是怕推送限流有点严重)
		var lastId int64 = 0 // 游标初始值

		for {
			// 获取一批 studentIds
			studentIds, newLastId, err := s.userFeedConfigDAO.GetStudentIdsByCursor(ctx, lastId, batchSize)
			if err != nil {
				s.l.Error("获取用户studentIds错误", append(logger.FormatLog("dao", err), logger.Int64("当前索引:", lastId))...)
			}

			// 如果没有更多数据，结束循环
			if len(studentIds) == 0 {
				return nil
			}

			// 遍历每个学生的 tokens
			for i := range studentIds {
				//更改id并推送
				event.StudentId = studentIds[i]
				err := s.feedProducer.SendMessage(topic.FeedEvent, event)
				if err != nil {
					s.l.Error("发送消息发生失败", append(logger.FormatLog("dao", err), logger.String("当前学号:", studentIds[i]))...)
				}
			}

			// 更新游标为最新值
			lastId = newLastId
		}
	}

	err := s.feedProducer.SendMessage(topic.FeedEvent, event)
	if err != nil {
		return PUBLIC_FEED_EVENT_ERROR(fmt.Errorf("%v,当前学号:%s", err, event.StudentId))
	}
	return nil
}

//func (s *feedEventService) insertEvent(ctx context.Context, feedEvent *model.FeedEvent, Type string) error {
//	// 开始事务，通过 DAO 层进行
//	tx, err := s.feedEventDAO.BeginTx(ctx)
//	if err != nil {
//		return err
//	}
//	defer func() {
//		if err != nil {
//			tx.Rollback()
//		} else {
//			tx.Commit()
//		}
//	}()
//
//	//插入feed数据到指定的表
//	insertedEvent, err := s.feedEventDAO.InsertFeedEvent(ctx, feedEvent)
//	if err != nil {
//		return err
//	}
//
//	//存储索引到指定的用户表
//	err = s.feedEventIndexDAO.SaveFeedEventIndex(ctx, &model.FeedEventIndex{
//		StudentId: feedEvent.StudentId,
//		FeedID:    insertedEvent.ID,
//		Read:      false,
//		Type:      Type,
//	})
//	if err != nil {
//		return err
//	}
//
//	return nil
//}
//
//func (s *feedEventService) insertEventsByType(ctx context.Context, Type string, events []model.FeedEvent) (err error) {
//	//开始事务
//	tx, err := s.feedEventDAO.BeginTx(ctx)
//	if err != nil {
//		return err
//	}
//
//	//批量存储
//	defer func() {
//		if err != nil {
//			tx.Rollback()
//		} else {
//			tx.Commit()
//		}
//	}()
//
//	// 插入 FeedEvent 并获取插入后的 ID
//	insertedEvents, err := s.feedEventDAO.InsertFeedEventListByTx(ctx, tx, events)
//	if err != nil {
//		return err
//	}
//
//	var eventIndexes []model.FeedEventIndex
//	// 根据插入后的 FeedEvent 生成 FeedEventIndex
//	for _, insertedEvent := range insertedEvents {
//		eventIndexes = append(eventIndexes, model.FeedEventIndex{
//			FeedID:    insertedEvent.ID,
//			StudentId: insertedEvent.StudentId,
//			Read:      false,
//			Type:      Type,
//		})
//	}
//
//	//批量插入
//	err = s.feedEventIndexDAO.InsertFeedEventIndexListByTx(ctx, tx, eventIndexes)
//	if err != nil {
//		return err
//	}
//
//	return nil
//}
