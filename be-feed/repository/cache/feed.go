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
	"github.com/redis/go-redis/v9"
)

//go:embed getPublicFeed.lua
var getPublicFeedLua string

//go:embed delFeed.lua
var delFeedLua string

type FeedEventCache interface {
	GetFeedEvent(ctx context.Context, feedType string, key string) (*model.FeedEvent, error)
	SetFeedEvent(ctx context.Context, durationTime time.Duration, key string, feedType string, feedEvent *model.FeedEvent) error
	//muxiEvents使用hashmap存，zset只存id
	SetMuxiFeeds(ctx context.Context, feedEvent MuxiOfficialMSG, publicTime int64) error
	GetMuxiToBePublicFeeds(ctx context.Context) ([]MuxiOfficialMSG, error)
	DelMuxiFeeds(ctx context.Context, id string) error
	ClearCache(ctx context.Context, feedType string, key string) error
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
	fullKey := cache.getKey(feedType, key)

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
	fullKey := cache.getKey(feedType, key)
	data, err := json.Marshal(*feedEvent)
	if err != nil {
		return err
	}
	return cache.cmd.Set(ctx, fullKey, data, durationTime).Err()
}

// 直接在redis层筛选到期要发布的feed，应用层就直接发布不需要筛选
func (cache *RedisFeedEventCache) GetMuxiToBePublicFeeds(ctx context.Context) ([]MuxiOfficialMSG, error) {
	zsetKey := cache.getPublicScoreKey()
	prefix := cache.getPrefix("muxi")
	var msgs []MuxiOfficialMSG
	results, err := cache.cmd.Eval(ctx, getPublicFeedLua, []string{zsetKey, prefix}, time.Now().Unix()).Result()
	if err != nil {
		return []MuxiOfficialMSG{}, err
	}
	//必须分两步类型转换，一步到位会报错
	resultsArr, ok := results.([]interface{})
	if !ok {
		return msgs, errors.New("格式不符")
	}

	for _, result := range resultsArr {
		data, ok := result.(string)
		if !ok {
			continue
		}
		var msg MuxiOfficialMSG
		err := json.Unmarshal([]byte(data), &msg)
		if err != nil {
			return []MuxiOfficialMSG{}, err
		}
		msgs = append(msgs, msg)
	}

	return msgs, nil

}

// 把publicTime从feedEvent中分离出来，作为score排序
func (cache *RedisFeedEventCache) SetMuxiFeeds(ctx context.Context, feedEvent MuxiOfficialMSG, publicTime int64) error {
	key := cache.getKey("muxi", feedEvent.MuxiMSGId)
	publicScoreKey := cache.getPublicScoreKey()

	//把feedEvent存入redis
	data, err := json.Marshal(feedEvent)
	if err != nil {
		return err
	}

	err = cache.cmd.Set(ctx, key, data, -1).Err()
	if err != nil {
		return err
	}
	//存入zset中
	err = cache.cmd.ZAdd(ctx, publicScoreKey, redis.Z{Member: []byte(feedEvent.MuxiMSGId), Score: float64(publicTime)}).Err()
	if err != nil {
		//回滚刚才的操作
		cache.cmd.Del(ctx, key)
		return err
	}
	return nil
}

func (cache *RedisFeedEventCache) DelMuxiFeeds(ctx context.Context, id string) error {
	key := cache.getKey("muxi", id)
	publicScoreKey := cache.getPublicScoreKey()
	_, err := cache.cmd.Eval(ctx, delFeedLua, []string{publicScoreKey, key}).Result()
	if err != nil {
		return err
	}
	return nil

}

func (cache *RedisFeedEventCache) ClearCache(ctx context.Context, feedType string, key string) error {
	// 生成带前缀的完整key
	fullKey := cache.getKey(feedType, key)
	return cache.cmd.Del(ctx, fullKey).Err()
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

	// 使用纳秒时间戳
	data := fmt.Sprintf("%d", time.Now().UnixNano())
	// 计算 SHA-256 哈希以确保唯一性
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

type MuxiOfficialMSG struct {
	MuxiMSGId          string //使用获取的uniqueId作为Id,防止误删
	Title              string
	Content            string
	model.ExtendFields //拓展字段如果要发额外的东西的话
	//PublicTime         int64 //正式发布的时间
}
