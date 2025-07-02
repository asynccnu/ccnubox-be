package biz

import (
	"context"

	"github.com/google/wire"
	"gorm.io/gorm"
)

// biz = domain + usecase
// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(NewLibraryUsecase)

// 公共依赖接口
type Transaction interface {
	InTx(ctx context.Context, fn func(context.Context) error) error
	DB(ctx context.Context) *gorm.DB
}

type CCNUServiceProxy interface {
	// 从其他服务获取cookie
	GetCookie(ctx context.Context, stuID string) (string, error)
}

type DelayQueue interface {
	Send(key, value []byte) error
	Consume(groupID string, f func(key, value []byte)) error
	Close()
}
