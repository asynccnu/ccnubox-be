package data

import (
	"context"
	"github.com/asynccnu/ccnubox-be/be-class/internal/conf"
	"github.com/redis/go-redis/v9"
	"time"
)

func NewRedisClient(cf *conf.Data) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cf.Redis.Addr,
		Password: cf.Redis.Password,
	})
}

type Cache struct {
	cli *redis.Client
}

func NewCache(cli *redis.Client) *Cache {
	return &Cache{cli: cli}
}

func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	val, err := c.cli.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}
	return val, nil
}

func (c *Cache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	err := c.cli.Set(ctx, key, value, expiration).Err()
	if err != nil {
		return err
	}
	return nil
}

func (c *Cache) Del(ctx context.Context, key ...string) error {
	if len(key) == 0 {
		return nil
	}
	return c.cli.Del(ctx, key...).Err()
}
