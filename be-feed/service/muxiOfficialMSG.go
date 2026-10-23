package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/asynccnu/ccnubox-be/be-feed/domain"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/cache"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/model"
	feedv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/feed/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// 定义错误结构体
var (
	GET_MUXI_FEED_ERROR    = errorx.FormatErrorFunc(feedv1.ErrorGetMuxiFeedError("获取木犀消息失败"))
	INSERT_MUXI_FEED_ERROR = errorx.FormatErrorFunc(feedv1.ErrorInsertMuxiFeedError("插入木犀消息失败"))
	REMOVE_MUXI_FEED_ERROR = errorx.FormatErrorFunc(feedv1.ErrorRemoveMuxiFeedError("删除木犀消息失败"))
)

type MuxiOfficialMSGService interface {
	SetOfficialWhiteList(ctx context.Context, list domain.OfficialWhiteList) (int, error)
	GetToBePublicOfficialMSG(ctx context.Context, isToPublic bool) ([]domain.MuxiOfficialMSG, error)
	PublicMuxiOfficialMSG(ctx context.Context, msg *domain.MuxiOfficialMSG) error
	StopMuxiOfficialMSG(ctx context.Context, id string) error
}

const officialWhiteListTTL = 7 * 24 * time.Hour
const officialWhiteListClockTolerance = time.Minute

func validateOfficialWhiteList(list domain.OfficialWhiteList, now time.Time) error {
	if list.Strategy == "" || list.ExpiresAt <= now.Unix() || list.ExpiresAt > now.Add(officialWhiteListTTL+officialWhiteListClockTolerance).Unix() {
		return status.Error(codes.FailedPrecondition, "请先通过 setWhiteList 设置有效名单")
	}
	if list.All {
		if list.Strategy != "all" || len(list.StudentIDs) != 0 {
			return status.Error(codes.InvalidArgument, "全量策略不能同时指定学号")
		}
		return nil
	}
	if len(list.StudentIDs) == 0 || (list.Strategy != "all" && len(list.StudentIDs) > 10000) {
		return status.Error(codes.InvalidArgument, "名单人数不合法")
	}
	seen := make(map[string]bool, len(list.StudentIDs))
	for _, id := range list.StudentIDs {
		if id == "" || strings.TrimSpace(id) != id || !utf8.ValidString(id) || utf8.RuneCountInString(id) > 255 || strings.ContainsAny(id, "\x00\r\n\t ") || seen[id] {
			return status.Error(codes.InvalidArgument, "名单包含非法或重复学号")
		}
		seen[id] = true
	}
	return nil
}

func (s *muxiOfficialMSGService) SetOfficialWhiteList(ctx context.Context, list domain.OfficialWhiteList) (int, error) {
	now := s.now()
	if err := validateOfficialWhiteList(list, now); err != nil {
		return 0, err
	}
	if list.Strategy == "all" {
		// Resolve on the service that owns Feed user registrations; never turn the
		// operator's all-current-students choice into an unrestricted broadcast.
		if !list.All {
			return 0, status.Error(codes.InvalidArgument, "全量名单必须由后端筛选")
		}
		ids, err := s.currentStudentIDs(ctx, now)
		if err != nil {
			return 0, err
		}
		list.StudentIDs = ids
		list.All = false
		if len(ids) == 0 {
			return 0, status.Error(codes.FailedPrecondition, "当前没有符合在校规则的用户")
		}
	}
	if err := s.feedEventCache.SetOfficialWhiteList(ctx, list); err != nil {
		return 0, err
	}
	return len(list.StudentIDs), nil
}

func (s *muxiOfficialMSGService) currentStudentIDs(ctx context.Context, at time.Time) ([]string, error) {
	const pageSize = 500
	var cursor int64
	seen := make(map[string]bool)
	ids := make([]string, 0)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		batch, next, err := s.userFeedConfigDAO.GetStudentIdsByCursor(ctx, cursor, pageSize)
		if err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}
		if next <= cursor {
			return nil, errors.New("名单分页游标未向前推进")
		}
		for _, id := range batch {
			if domain.IsCurrentStudent(id, at) && !seen[id] {
				ids = append(ids, id)
				seen[id] = true
			}
		}
		cursor = next
	}
	sort.Strings(ids)
	return ids, nil
}

type muxiOfficialMSGService struct {
	now               func() time.Time
	feedEventDAO      dao.FeedEventDAO
	feedEventCache    cache.FeedEventCache
	userFeedConfigDAO dao.FeedUserConfigDAO
	muxiRedisLock     *sync.Mutex // 使用指针以确保实例唯一性
}

func NewMuxiOfficialMSGService(feedEventDAO dao.FeedEventDAO, feedEventCache cache.FeedEventCache, feedAllowListEventDAO dao.FeedUserConfigDAO) MuxiOfficialMSGService {
	return &muxiOfficialMSGService{
		now:               time.Now,
		feedEventCache:    feedEventCache,
		feedEventDAO:      feedEventDAO,
		userFeedConfigDAO: feedAllowListEventDAO,
		muxiRedisLock:     &sync.Mutex{},
	}
}

// isToPublic:获取要发送的feedEvent；!isToPublic:获取还未发送的消息
func (s *muxiOfficialMSGService) GetToBePublicOfficialMSG(ctx context.Context, isToPublic bool) ([]domain.MuxiOfficialMSG, error) {
	feeds, err := s.feedEventCache.GetMuxiToBePublicFeeds(ctx, isToPublic)
	if err != nil {
		return nil, GET_MUXI_FEED_ERROR(errorx.Errorf("service: get muxi feeds from cache failed, isToPublic: %v, err: %w", isToPublic, err))
	}

	return convMuxiMessageFromCacheToDomain(feeds), nil
}

func (s *muxiOfficialMSGService) PublicMuxiOfficialMSG(ctx context.Context, msg *domain.MuxiOfficialMSG) error {
	s.muxiRedisLock.Lock()
	defer s.muxiRedisLock.Unlock()
	list, err := s.feedEventCache.GetOfficialWhiteList(ctx)
	if errors.Is(err, redis.Nil) {
		return status.Error(codes.FailedPrecondition, "请先通过 setWhiteList 设置发送名单")
	}
	if err != nil {
		return err
	}
	if err := validateOfficialWhiteList(list, s.now()); err != nil {
		return err
	}

	if list.All {
		ids, err := s.currentStudentIDs(ctx, s.now())
		if err != nil {
			return err
		}
		list.StudentIDs = ids
		list.All = false
	}
	if list.Strategy == "all" {
		at := s.now()
		if msg.PublicTime > at.Unix() {
			at = time.Unix(msg.PublicTime, 0)
		}
		filtered := make([]string, 0, len(list.StudentIDs))
		for _, id := range list.StudentIDs {
			if domain.IsCurrentStudent(id, at) {
				filtered = append(filtered, id)
			}
		}
		if len(filtered) == 0 {
			return status.Error(codes.FailedPrecondition, "名单中已没有符合在校规则的用户，请重新设置名单")
		}
		list.StudentIDs = filtered
	}

	uniqueKey := s.feedEventCache.GetUniqueKey()
	feed := cache.MuxiOfficialMSG{
		Targeted:          true,
		WhiteListStrategy: list.Strategy,
		StudentIDs:        append([]string(nil), list.StudentIDs...),
		MuxiMSGId:         uniqueKey,
		Title:             msg.Title,
		Content:           msg.Content,
		ExtendFields:      model.ExtendFields(msg.ExtendFields),
	}

	err = s.feedEventCache.SetMuxiFeeds(ctx, feed, msg.PublicTime)
	if err != nil {
		return INSERT_MUXI_FEED_ERROR(errorx.Errorf("service: set muxi feeds to cache failed, title: %s, msgId: %s, err: %w", msg.Title, uniqueKey, err))
	}

	return nil
}

func (s *muxiOfficialMSGService) StopMuxiOfficialMSG(ctx context.Context, MSGId string) error {
	s.muxiRedisLock.Lock()
	defer s.muxiRedisLock.Unlock()

	err := s.feedEventCache.DelMuxiFeeds(ctx, MSGId)
	if err != nil {
		return REMOVE_MUXI_FEED_ERROR(errorx.Errorf("service: delete muxi feeds from cache failed, msgId: %s, err: %w", MSGId, err))
	}
	return nil
}
