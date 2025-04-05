package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/asynccnu/ccnubox-be/be-feed/repository/model"
	"github.com/redis/go-redis/v9"
	"time"
)

type FeedEventCache interface {
	GetFeedEvent(ctx context.Context, feedType string, key string) (*model.FeedEvent, error)
	SetFeedEvent(ctx context.Context, durationTime time.Duration, key string, feedType string, feedEvent *model.FeedEvent) error
	SetMuxiFeeds(ctx context.Context, feedEvent []MuxiOfficialMSG) error
	GetMuxiFeeds(ctx context.Context) ([]MuxiOfficialMSG, error)
	ClearCache(ctx context.Context, key string) error
	GetUniqueKey() string
}

type RedisFeedEventCache struct {
	cmd redis.Cmdable
}

// 基本完成,现在唯一没做的就是对于推送给全体用户的消息没有从缓存中获取(可以优化但是目前我的执行方案链路和层次都太复杂了,目前打算暂时放弃)
func NewRedisFeedEventCache(cmd redis.Cmdable) FeedEventCache {
	return &RedisFeedEventCache{cmd: cmd}
}

func (cache *RedisFeedEventCache) GetFeedEvent(ctx context.Context, feedType string, key string) (*model.FeedEvent, error) {
	//使用前缀加上唯一索引的方式存储到缓存
	fullKey := cache.getKey(feedType + key)

	data, err := cache.cmd.Get(ctx, fullKey).Bytes()
	if err != nil {
		return &model.FeedEvent{}, err
	}
	var st model.FeedEvent
	err = json.Unmarshal(data, &st)
	return &st, err
}

func (cache *RedisFeedEventCache) SetFeedEvent(ctx context.Context, durationTime time.Duration, key string, feedType string, feedEvent *model.FeedEvent) error {
	//使用前缀加上唯一索引的方式存储到缓存
	fullKey := cache.getKey(feedType + key)
	data, err := json.Marshal(*feedEvent)
	if err != nil {
		return err
	}
	return cache.cmd.Set(ctx, fullKey, data, durationTime).Err()
}

func (cache *RedisFeedEventCache) GetMuxiFeeds(ctx context.Context) ([]MuxiOfficialMSG, error) {
	key := cache.getKey("muxi")
	data, err := cache.cmd.Get(ctx, key).Bytes()
	switch err {
	case redis.Nil:
		return []MuxiOfficialMSG{}, nil
	case nil:
	default:
		return []MuxiOfficialMSG{}, err

	}
	var st []MuxiOfficialMSG
	err = json.Unmarshal(data, &st)
	return st, err
}

func (cache *RedisFeedEventCache) SetMuxiFeeds(ctx context.Context, feedEvent []MuxiOfficialMSG) error {
	key := cache.getKey("muxi")
	data, err := json.Marshal(feedEvent)
	if err != nil {
		return err
	}
	return cache.cmd.Set(ctx, key, data, 0).Err()
}

func (cache *RedisFeedEventCache) ClearCache(ctx context.Context, key string) error {
	// 生成带前缀的完整key
	fullKey := cache.getKey(key)
	return cache.cmd.Del(ctx, fullKey).Err()
}

func (cache *RedisFeedEventCache) getKey(value string) string {
	return "ccnubox:feed:" + value
}

func (cache *RedisFeedEventCache) GetUniqueKey() string {

	// 使用纳秒时间戳
	data := fmt.Sprintf("%d", time.Now().UnixNano())
	// 计算 SHA-256 哈希以确保唯一性
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

type MuxiOfficialMSG struct {
	MuixMSGId          string //使用获取的uniqueId作为Id,防止误删
	Title              string
	Content            string
	model.ExtendFields       //拓展字段如果要发额外的东西的话
	PublicTime         int64 //正式发布的时间
}
