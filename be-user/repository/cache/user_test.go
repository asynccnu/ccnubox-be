package cache

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/asynccnu/ccnubox-be/common/tool"
	"github.com/redis/go-redis/v9"
)

type cookieRedis struct {
	redis.Cmdable
	values  map[string]string
	ttls    map[string]time.Duration
	deleted []string
}

func (r *cookieRedis) Get(ctx context.Context, key string) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx)
	if value, ok := r.values[key]; ok {
		cmd.SetVal(value)
	} else {
		cmd.SetErr(redis.Nil)
	}
	return cmd
}

func (r *cookieRedis) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) *redis.StatusCmd {
	r.values[key] = value.(string)
	r.ttls[key] = ttl
	cmd := redis.NewStatusCmd(ctx)
	cmd.SetVal("OK")
	return cmd
}

func (r *cookieRedis) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	r.deleted = append(r.deleted, keys...)
	for _, key := range keys {
		delete(r.values, key)
	}
	return redis.NewIntCmd(ctx)
}

func TestCookieKeysAreIsolatedAndIgnoreLegacyCache(t *testing.T) {
	ctx := context.Background()
	r := &cookieRedis{values: map[string]string{"ccnubox:users:xk:2024200000": "legacy"}, ttls: make(map[string]time.Duration)}
	c := &RedisUserCache{cmd: r}
	for _, system := range []tool.XKSystem{tool.XKUndergraduate, tool.XKPostgraduate} {
		if _, err := c.GetCookie(ctx, "2024200000", system); err != ErrKeyNotFound {
			t.Fatalf("system=%s read legacy cookie: %v", system, err)
		}
		if err := c.SetCookie(ctx, "2024200000", "cookie-"+string(system), system); err != nil {
			t.Fatal(err)
		}
	}
	for _, system := range []tool.XKSystem{tool.XKUndergraduate, tool.XKPostgraduate} {
		cookie, err := c.GetCookie(ctx, "2024200000", system)
		if err != nil || cookie != "cookie-"+string(system) {
			t.Fatalf("system=%s cookie=%q err=%v", system, cookie, err)
		}
		if r.ttls[c.cookieKey("2024200000", system)] != 5*time.Minute {
			t.Error("cookie TTL changed")
		}
	}
	if _, err := c.GetCookie(ctx, "2024200001", tool.XKUndergraduate); err != ErrKeyNotFound {
		t.Fatalf("different student shared cookie: %v", err)
	}
}

func TestDeleteUserDataRemovesNewAndLegacyKeys(t *testing.T) {
	keys := []string{
		"ccnubox:users:xk:2024200000",
		"ccnubox:users:xk:v2:undergraduate:2024200000",
		"ccnubox:users:xk:v2:postgraduate:2024200000",
		"ccnubox:users:lib:LIBRARY_SEAT:2024200000",
		"ccnubox:users:lib:LIBRARY_DISCUSSION:2024200000",
	}
	r := &cookieRedis{values: map[string]string{"unrelated": "keep"}}
	for _, key := range keys {
		r.values[key] = "test-value"
	}
	c := &RedisUserCache{cmd: r}
	if err := c.DeleteUserData(context.Background(), "2024200000"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r.deleted, keys) || len(r.values) != 1 || r.values["unrelated"] != "keep" {
		t.Fatalf("deleted=%v remaining=%v", r.deleted, r.values)
	}
}
