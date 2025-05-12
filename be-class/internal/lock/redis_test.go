package lock

import (
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRedisLock(t *testing.T) {
	cli := redis.NewClient(&redis.Options{
		Addr: "localhost:16379",
	})
	builder := NewRedisLockBuilder(cli)
	locker1 := builder.Build("test")
	locker2 := builder.Build("test")

	t.Run("the lock is locked", func(t *testing.T) {
		err := locker1.Lock()
		assert.NoError(t, err)
		err = locker2.Lock()
		assert.Error(t, err)
		ok, err := locker1.Unlock()
		assert.NoError(t, err)
		assert.True(t, ok)
	})
	t.Run("the lock is unlocked", func(t *testing.T) {
		err := locker1.Lock()
		assert.NoError(t, err)
		ok, err := locker1.Unlock()
		assert.NoError(t, err)
		assert.True(t, ok)
		err = locker2.Lock()
		assert.NoError(t, err)
		ok, err = locker2.Unlock()
		assert.NoError(t, err)
		assert.True(t, ok)
	})
}
