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
	creditPointsKeyPrefix = "lib:credit:point:"
	creditPointsTTL       = 60 * time.Second
)

type CreditPointCache struct {
	redis *redis.Client
	log   *log.Helper
}

func NewCreditPointCache(redis *redis.Client, logger log.Logger) *CreditPointCache {
	return &CreditPointCache{
		redis: redis,
		log:   log.NewHelper(logger),
	}
}

func (c *CreditPointCache) creditPointKey(stuID string) string {
	return fmt.Sprintf("%s%s", creditPointsKeyPrefix, stuID)
}

func (c *CreditPointCache) Get(ctx context.Context, stuID string) (*biz.CreditPoints, bool, error) {
	key := c.creditPointKey(stuID)
	val, err := c.redis.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var out *biz.CreditPoints
	if err = json.Unmarshal(val, &out); err != nil {
		return nil, false, err
	}
	return out, true, nil
}

func (c *CreditPointCache) Set(ctx context.Context, stuID string, data *biz.CreditPoints) error {
	key := c.creditPointKey(stuID)
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.redis.Set(ctx, key, jsonData, creditPointsTTL).Err()
}

func (c *CreditPointCache) Del(ctx context.Context, stuID string) error {
	key := c.creditPointKey(stuID)
	return c.redis.Del(ctx, key).Err()
}
