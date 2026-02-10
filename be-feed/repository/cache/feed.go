package cache

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/be-feed/repository/model"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/redis/go-redis/v9"
)

//go:embed getPublicFeed.lua
var getPublicFeedLua string

//go:embed delFeed.lua
var delFeedLua string

type FeedEventCache interface {
	GetFeedEvent(ctx context.Context, feedType string, key string) (*model.FeedEvent, error)
	SetFeedEvent(ctx context.Context, durationTime time.Duration, key string, feedType string, feedEvent *model.FeedEvent) error
	SetMuxiFeeds(ctx context.Context, feedEvent MuxiOfficialMSG, publicTime int64) error
	GetMuxiToBePublicFeeds(ctx context.Context, isToPublic bool) ([]MuxiOfficialMSG, error)
	DelMuxiFeeds(ctx context.Context, id string) error
	ClearCache(ctx context.Context, feedType string, key string) error
	GetUniqueKey() string
}

type RedisFeedEventCache struct {
	cmd redis.Cmdable
}

func NewRedisFeedEventCache(cmd redis.Cmdable) FeedEventCache {
	return &RedisFeedEventCache{cmd: cmd}
}

func (cache *RedisFeedEventCache) GetFeedEvent(ctx context.Context, feedType string, key string) (*model.FeedEvent, error) {
	fullKey := cache.getKey(feedType, key)

	data, err := cache.cmd.Get(ctx, fullKey).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, err // 保持 redis.Nil 向上透传，由 service 处理命中逻辑
		}
		return nil, errorx.Errorf("cache: get feed event failed, key: %s, err: %w", fullKey, err)
	}

	var st model.FeedEvent
	if err = json.Unmarshal(data, &st); err != nil {
		return nil, errorx.Errorf("cache: unmarshal feed event failed, key: %s, err: %w", fullKey, err)
	}
	return &st, nil
}

func (cache *RedisFeedEventCache) SetFeedEvent(ctx context.Context, durationTime time.Duration, key string, feedType string, feedEvent *model.FeedEvent) error {
	fullKey := cache.getKey(feedType, key)
	data, err := json.Marshal(*feedEvent)
	if err != nil {
		return errorx.Errorf("cache: marshal feed event failed, key: %s, err: %w", fullKey, err)
	}

	err = cache.cmd.Set(ctx, fullKey, data, durationTime).Err()
	if err != nil {
		return errorx.Errorf("cache: set redis failed, key: %s, err: %w", fullKey, err)
	}
	return nil
}

func (cache *RedisFeedEventCache) GetMuxiToBePublicFeeds(ctx context.Context, isToPublic bool) ([]MuxiOfficialMSG, error) {
	zsetKey := cache.getPublicScoreKey()

	var msgs []MuxiOfficialMSG
	results, err := cache.cmd.Eval(ctx, getPublicFeedLua, []string{zsetKey}, time.Now().Unix(), isToPublic).Result()
	if err != nil {
		return nil, errorx.Errorf("cache: eval getPublicFeedLua failed, zsetKey: %s, err: %w", zsetKey, err)
	}

	resultsArr, ok := results.([]interface{})
	if !ok {
		return nil, errorx.Errorf("cache: unexpected lua result format, expected []interface{}, got %T", results)
	}

	for i, result := range resultsArr {
		data, ok := result.(string)
		if !ok {
			continue
		}
		var msg MuxiOfficialMSG
		if err := json.Unmarshal([]byte(data), &msg); err != nil {
			return nil, errorx.Errorf("cache: unmarshal muxi msg failed at index %d, data: %s, err: %w", i, data, err)
		}
		msgs = append(msgs, msg)
	}

	return msgs, nil
}

func (cache *RedisFeedEventCache) SetMuxiFeeds(ctx context.Context, feedEvent MuxiOfficialMSG, publicTime int64) error {
	key := feedEvent.MuxiMSGId
	publicScoreKey := cache.getPublicScoreKey()

	data, err := json.Marshal(feedEvent)
	if err != nil {
		return errorx.Errorf("cache: marshal muxi feed failed, msgId: %s, err: %w", key, err)
	}

	// 存入 String Key
	err = cache.cmd.Set(ctx, key, data, -1).Err()
	if err != nil {
		return errorx.Errorf("cache: set muxi string key failed, msgId: %s, err: %w", key, err)
	}

	// 存入 ZSet
	err = cache.cmd.ZAdd(ctx, publicScoreKey, redis.Z{Member: key, Score: float64(publicTime)}).Err()
	if err != nil {
		// 回退操作
		cache.cmd.Del(ctx, key)
		return errorx.Errorf("cache: zadd muxi feed failed, msgId: %s, scoreKey: %s, err: %w", key, publicScoreKey, err)
	}
	return nil
}

func (cache *RedisFeedEventCache) DelMuxiFeeds(ctx context.Context, id string) error {
	publicScoreKey := cache.getPublicScoreKey()
	_, err := cache.cmd.Eval(ctx, delFeedLua, []string{publicScoreKey, id}).Result()
	if err != nil {
		return errorx.Errorf("cache: eval delFeedLua failed, msgId: %s, scoreKey: %s, err: %w", id, publicScoreKey, err)
	}
	return nil
}

func (cache *RedisFeedEventCache) ClearCache(ctx context.Context, feedType string, key string) error {
	fullKey := cache.getKey(feedType, key)
	err := cache.cmd.Del(ctx, fullKey).Err()
	if err != nil {
		return errorx.Errorf("cache: del redis key failed, key: %s, err: %w", fullKey, err)
	}
	return nil
}

func (cache *RedisFeedEventCache) getPublicScoreKey() string {
	return "ccnubox:feed:toPublic"
}

func (cache *RedisFeedEventCache) getKey(types string, value string) string {
	return cache.getPrefix(types) + value
}

func (cache *RedisFeedEventCache) getPrefix(types string) string {
	return "ccnubox:feed:" + types + ":"
}

func (cache *RedisFeedEventCache) GetUniqueKey() string {
	data := fmt.Sprintf("%d", time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

type MuxiOfficialMSG struct {
	MuxiMSGId string `json:"muxi_msg_id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Url       string `json:"url"`
	model.ExtendFields
}
