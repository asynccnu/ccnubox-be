package biz

import (
	"context"
	"time"

	"github.com/google/wire"
)

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(NewClassServiceUserCase, NewFreeClassroomBiz)

type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Del(ctx context.Context, key ...string) error
	SAdd(ctx context.Context, key string, members ...interface{}) error
	SMembers(ctx context.Context, key string) ([]string, error)
	SExpire(ctx context.Context, key string, expire time.Duration) error
}

const (
	Finished = "finished"
	Failed   = "failed"
)
