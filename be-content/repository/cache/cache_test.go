package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/asynccnu/ccnubox-be/be-content/repository/model"
)

func TestCache(t *testing.T) {
	// 1. 启动内存 Redis
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis failed: %v", err)
	}
	defer mr.Close()

	// 2. 创建 redis client
	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// 3. 创建泛型缓存
	cache := NewRedisCache[model.Website](rdb)

	ctx := context.Background()

	// 4. 构造测试数据
	data := []model.Website{
		{
			Name: "Google",
		},
		{
			Name: "GitHub",
		},
	}

	// 5. 设置缓存
	if err := cache.SetContent(ctx, data, time.Minute); err != nil {
		t.Fatalf("SetContent failed: %v", err)
	}

	// 6. 读取缓存
	res, err := cache.GetContent(ctx)
	if err != nil {
		t.Fatalf("GetContent failed: %v", err)
	}

	// 7. 校验
	if len(res) != len(data) {
		t.Fatalf("length mismatch: %d != %d", len(res), len(data))
	}

	if res[0].Name != "Google" {
		t.Fatalf("unexpected value: %+v", res[0])
	}

	// 8. 清除缓存
	if err := cache.ClearContent(ctx); err != nil {
		t.Fatalf("ClearContent failed: %v", err)
	}

	// 9. 再次读取应失败
	_, err = cache.GetContent(ctx)
	if err == nil {
		t.Fatalf("expected error after clear, got nil")
	}
}
