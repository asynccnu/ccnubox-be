package service

import (
	"context"
	"sync"

	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/cache"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/model"
	feedv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feed/v1"
	errorx "github.com/asynccnu/ccnubox-be/common/pkg/errorx/rpcerr"
)

type MuxiOfficialMSGService interface {
	GetToBePublicOfficialMSG(ctx context.Context, isToPublic bool) ([]domain.MuxiOfficialMSG, error)
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
		return errorx.New(feedv1.ErrorRemoveMuxiFeedError("删除木犀消息失败"), "cache", err)
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

// isToPublic:获取要发送的feedEvent；!isToPublic:获取还未发送的消息
func (s *muxiOfficialMSGService) GetToBePublicOfficialMSG(ctx context.Context, isToPublic bool) ([]domain.MuxiOfficialMSG, error) {
	feeds, err := s.feedEventCache.GetMuxiToBePublicFeeds(ctx, isToPublic)
	if err != nil {
		return nil, GET_MUXI_FEED_ERROR(err)
	}

	return convMuxiMessageFromCacheToDomain(feeds), nil
}

func (s *muxiOfficialMSGService) PublicMuxiOfficialMSG(ctx context.Context, msg *domain.MuxiOfficialMSG) error {
	s.muxiRedisLock.Lock()
	defer s.muxiRedisLock.Unlock()

	feed := cache.MuxiOfficialMSG{
		MuxiMSGId:    s.feedEventCache.GetUniqueKey(),
		Title:        msg.Title,
		Content:      msg.Content,
		ExtendFields: model.ExtendFields(msg.ExtendFields),
	}

	err := s.feedEventCache.SetMuxiFeeds(ctx, feed, msg.PublicTime)
	if err != nil {
		return INSERT_MUXI_FEED_ERROR(err)
	}

	return nil
}

func (s *muxiOfficialMSGService) StopMuxiOfficialMSG(ctx context.Context, MSGId string) error {
	s.muxiRedisLock.Lock()
	defer s.muxiRedisLock.Unlock()

	err := s.feedEventCache.DelMuxiFeeds(ctx, MSGId)
	if err != nil {
		return REMOVE_MUXI_FEED_ERROR(err)
	}
	return nil
}
