package data

import (
	"context"
	"time"

	"github.com/asynccnu/ccnubox-be/be-class/internal/conf"
	"github.com/redis/go-redis/v9"
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

func (c *Cache) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return c.cli.SAdd(ctx, key, members...).Err()
}

func (c *Cache) SMembers(ctx context.Context, key string) ([]string, error) {
	members, err := c.cli.SMembers(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	return members, nil
}

func (c *Cache) SExpire(ctx context.Context, key string, expire time.Duration) error {
	return c.cli.Expire(ctx, key, expire).Err()
}
