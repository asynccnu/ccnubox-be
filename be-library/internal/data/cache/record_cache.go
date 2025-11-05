package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/be-library/internal/biz"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
)

const (
	futureRecordKeyPrefix  = "lib:future:records:"
	futureRecordTTL        = 60 * time.Second
	historyRecordKeyPrefix = "lib:history:records:"
	historyRecordTTL       = 60 * time.Second
)

type RecordCache struct {
	redis *redis.Client
	log   *log.Helper
}

func NewRecordCache(redis *redis.Client, logger log.Logger) *RecordCache {
	return &RecordCache{
		redis: redis,
		log:   log.NewHelper(logger),
	}
}

func (c *RecordCache) futureRecordKey(stuID string) string {
	return fmt.Sprintf("%s%s", futureRecordKeyPrefix, stuID)
}

func (c *RecordCache) GetFutureRecords(ctx context.Context, stuID string) ([]*biz.FutureRecords, bool, error) {
	key := c.futureRecordKey(stuID)
	val, err := c.redis.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var out []*biz.FutureRecords
	if err = json.Unmarshal(val, &out); err != nil {
		return nil, false, err
	}
	return out, true, nil
}

func (c *RecordCache) SetFutureRecords(ctx context.Context, stuID string, list []*biz.FutureRecords) error {
	key := c.futureRecordKey(stuID)
	data, err := json.Marshal(list)
	if err != nil {
		return err
	}
	return c.redis.Set(ctx, key, data, futureRecordTTL).Err()
}

func (c *RecordCache) DelFutureRecords(ctx context.Context, stuID string) error {
	key := c.futureRecordKey(stuID)
	return c.redis.Del(ctx, key).Err()
}

func (c *RecordCache) historyRecordKey(stuID string) string {
	return fmt.Sprintf("%s%s", historyRecordKeyPrefix, stuID)
}

func (c *RecordCache) GetHistoryRecords(ctx context.Context, stuID string) ([]*biz.HistoryRecords, bool, error) {
	key := c.historyRecordKey(stuID)
	val, err := c.redis.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var out []*biz.HistoryRecords
	if err = json.Unmarshal(val, &out); err != nil {
		return nil, false, err
	}
	return out, true, nil
}

func (c *RecordCache) SetHistoryRecords(ctx context.Context, stuID string, list []*biz.HistoryRecords) error {
	key := c.historyRecordKey(stuID)
	data, err := json.Marshal(list)
	if err != nil {
		return err
	}
	return c.redis.Set(ctx, key, data, historyRecordTTL).Err()
}

func (c *RecordCache) DelHistoryRecords(ctx context.Context, stuID string) error {
	key := c.historyRecordKey(stuID)
	return c.redis.Del(ctx, key).Err()
}
