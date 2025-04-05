package service

import (
	"context"
	"errors"
	feedv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/feed/v1"
	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	"github.com/asynccnu/ccnubox-be/be-feed/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/cache"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/model"
	"github.com/redis/go-redis/v9"
	"sync"
)

type MuxiOfficialMSGService interface {
	GetToBePublicOfficialMSG(ctx context.Context) ([]domain.MuxiOfficialMSG, error)
	GetMuxiOfficialMSGById(ctx context.Context, id string) (*domain.MuxiOfficialMSG, error)
	PublicMuxiOfficialMSG(ctx context.Context, msg *domain.MuxiOfficialMSG) error
	StopMuxiOfficialMSG(ctx context.Context, id string) error
}

// 定义错误结构体
var (
	GET_MUXI_FEED_ERROR = func(err error) error {
		return errorx.New(feedv1.ErrorGetMuxiFeedError("获取木犀消息失败"), "cache", err)
	}

	INSERT_MUXI_FEED_ERROR = func(err error) error {
		return errorx.New(feedv1.ErrorInsertMuxiFeedError("插入木犀消息失败"), "cache", err)
	}

	REMOVE_MUXI_FEED_ERROR = func(err error) error {
		return errorx.New(feedv1.ErrorRemoveMuxiFeedError("删除木犀消息"), "cache", err)
	}
)

type muxiOfficialMSGService struct {
	feedEventDAO      dao.FeedEventDAO
	feedEventCache    cache.FeedEventCache
	userFeedConfigDAO dao.UserFeedConfigDAO
	muxiRedisLock     sync.Mutex //用于防止读取muxi缓存不一致
}

func NewMuxiOfficialMSGService(feedEventDAO dao.FeedEventDAO, feedEventCache cache.FeedEventCache, feedAllowListEventDAO dao.UserFeedConfigDAO) MuxiOfficialMSGService {
	return &muxiOfficialMSGService{
		feedEventCache:    feedEventCache,
		feedEventDAO:      feedEventDAO,
		userFeedConfigDAO: feedAllowListEventDAO,
		muxiRedisLock:     sync.Mutex{},
	}
}

func (s *muxiOfficialMSGService) GetToBePublicOfficialMSG(ctx context.Context) ([]domain.MuxiOfficialMSG, error) {
	feeds, err := s.feedEventCache.GetMuxiFeeds(ctx)
	if err != nil {
		return nil, GET_MUXI_FEED_ERROR(err)
	}

	return convMuxiMessageFromCacheToDomain(feeds), nil
}

func (s *muxiOfficialMSGService) GetMuxiOfficialMSGById(ctx context.Context, id string) (*domain.MuxiOfficialMSG, error) {
	feeds, err := s.feedEventCache.GetMuxiFeeds(ctx)
	if err != nil {
		return &domain.MuxiOfficialMSG{}, GET_MUXI_FEED_ERROR(err)
	}

	for _, feed := range feeds {
		if feed.MuixMSGId == id {
			//返回结果
			return &domain.MuxiOfficialMSG{
				Title:        feed.Title,
				Content:      feed.Content,
				ExtendFields: domain.ExtendFields(feed.ExtendFields),
				PublicTime:   feed.PublicTime,
				Id:           id,
			}, nil
		}
	}
	return &domain.MuxiOfficialMSG{}, GET_MUXI_FEED_ERROR(errors.New("无法找到指定muxi消息"))
}

func (s *muxiOfficialMSGService) PublicMuxiOfficialMSG(ctx context.Context, msg *domain.MuxiOfficialMSG) error {
	s.muxiRedisLock.Lock()
	defer s.muxiRedisLock.Unlock()

	feeds, err := s.feedEventCache.GetMuxiFeeds(ctx)
	switch err {
	case redis.Nil:
		//如果不存在这个键的话设置为空
		feeds = []cache.MuxiOfficialMSG{}
	case nil:
		//正常
	default:
		return GET_MUXI_FEED_ERROR(err)
	}

	feeds = append(feeds, cache.MuxiOfficialMSG{
		MuixMSGId:    s.feedEventCache.GetUniqueKey(), //设置唯一id
		Title:        msg.Title,
		Content:      msg.Content,
		ExtendFields: model.ExtendFields(msg.ExtendFields),
		PublicTime:   msg.PublicTime, //获取将要发表的时间
	})

	err = s.feedEventCache.SetMuxiFeeds(ctx, feeds)
	if err != nil {
		return INSERT_MUXI_FEED_ERROR(err)
	}

	return nil
}

func (s *muxiOfficialMSGService) StopMuxiOfficialMSG(ctx context.Context, MSGId string) error {
	s.muxiRedisLock.Lock()
	defer s.muxiRedisLock.Unlock()

	feeds, err := s.feedEventCache.GetMuxiFeeds(ctx)
	if err != nil {
		return GET_MUXI_FEED_ERROR(err)
	}

	var newFeeds []cache.MuxiOfficialMSG
	for _, feed := range feeds {
		//只保留id不同的部分
		if feed.MuixMSGId != MSGId {
			newFeeds = append(newFeeds, feed)
		}
	}

	err = s.feedEventCache.SetMuxiFeeds(ctx, newFeeds)
	if err != nil {
		return REMOVE_MUXI_FEED_ERROR(err)
	}
	return nil
}
