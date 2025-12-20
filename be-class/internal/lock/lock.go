package lock

import (
	"time"

	"github.com/google/wire"
)

type Locker interface {
	Lock() error
	Unlock() (bool, error)
}

type Builder interface {
	Build(name string) Locker
	BuildWithExpire(name string, expire time.Duration) Locker
}

var ProviderSet = wire.NewSet(NewRedisLockBuilder)
