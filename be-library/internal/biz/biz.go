package biz

import (
	"context"
	"time"

	"github.com/asynccnu/ccnubox-be/be-library/internal/conf"
	"github.com/google/wire"
	"gorm.io/gorm"
)

// biz = domain + usecase
// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(NewLibraryBiz, NewWaitTime, NewCommentUsecase)

// NewWaitTime 提供等待时间配置
func NewWaitTime(cf *conf.Server) time.Duration {
	waitTime := 1200 * time.Millisecond

	if cf.Grpc.Timeout != nil && cf.Grpc.Timeout.Seconds > 0 {
		waitTime = cf.Grpc.Timeout.AsDuration()
	}

	return waitTime
}

// 公共依赖接口
type Transaction interface {
	InTx(ctx context.Context, fn func(context.Context) error) error
	DB(ctx context.Context) *gorm.DB
}

type CCNUServiceProxy interface {
	// 从其他服务获取cookie
	GetLibrarySeatToken(ctx context.Context, stuID string) (string, error)
	GetLibraryDiscussionToken(ctx context.Context, stuID string) (string, error)
}

type DelayQueue interface {
	Send(key, value []byte) error
	Consume(groupID string, f func(key, value []byte)) error
	Close()
}
