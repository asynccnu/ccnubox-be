package lock

import "github.com/google/wire"

type Locker interface {
	Lock() error
	Unlock() (bool, error)
}

type Builder interface {
	Build(name string) Locker
}

var ProviderSet = wire.NewSet(NewRedisLockBuilder)
