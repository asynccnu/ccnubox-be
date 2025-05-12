package biz

import (
	"context"
	"github.com/google/wire"
	"time"
)

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(NewClassSerivceUserCase, NewFreeClassroomBiz)

type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Del(ctx context.Context, key ...string) error
}

const (
	Finished = "finished"
	Failed   = "failed"
)
