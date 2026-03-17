package cache

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/conf"
	"github.com/asynccnu/ccnubox-be/be-classlist_v2/repository/model"
	com_conf "github.com/asynccnu/ccnubox-be/common/bizpkg/conf"
	"github.com/asynccnu/ccnubox-be/common/bizpkg/log"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/redis/go-redis/v9"
)

func initTestLogger(t *testing.T) {
	t.Helper()

	logDir := "./debug_log"
	logPath := filepath.Join(logDir, "test.log")

	cfg := &conf.ServerConf{
		BaseServerConf: com_conf.BaseServerConf{
			Log: &com_conf.LogConf{
				Path:       logPath,
				MaxSize:    1,
				MaxBackups: 1,
				MaxAge:     1,
				Compress:   false,
			},
		},
	}

	l := log.InitLogger(cfg.Log, 3)
	logger.InitGlobalLogger(l)
}

func newRepoWithLogger(t *testing.T) (*ClassInfoCache, *miniredis.Miniredis) {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis failed: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	cfg := &conf.ServerConf{
		ClassListConf: &conf.ClassListConf{
			ClassExpiration:     60,
			BlackListExpiration: 5,
		},
	}

	initTestLogger(t)
	return NewClassInfoCache(BaseCache{rdb: rdb}, cfg), mr
}

func TestClassInfoCache_SetGet(t *testing.T) {
	repo, mr := newRepoWithLogger(t)
	defer mr.Close()

	ctx := context.Background()

	data := []*model.ClassInfo{
		{ID: "1", Classname: "Math", Year: "2024", Semester: "1"},
	}

	if err := repo.AddClaInfosToCache(ctx, "sid", "2024", "1", data); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	got, err := repo.GetClassInfosFromCache(ctx, "sid", "2024", "1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(got) != 1 || got[0].ID != "1" {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestClassInfoCache_NullSentinel(t *testing.T) {
	repo, mr := newRepoWithLogger(t)
	defer mr.Close()

	ctx := context.Background()

	if err := repo.AddClaInfosToCache(ctx, "sid", "2024", "1", nil); err != nil {
		t.Fatalf("Add nil failed: %v", err)
	}

	got, err := repo.GetClassInfosFromCache(ctx, "sid", "2024", "1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil, got: %+v", got)
	}
}

func TestClassInfoCache_Delete(t *testing.T) {
	repo, mr := newRepoWithLogger(t)
	defer mr.Close()

	ctx := context.Background()
	_ = repo.AddClaInfosToCache(ctx, "sid", "2024", "1", []*model.ClassInfo{{ID: "1"}})

	if err := repo.DeleteClassInfoFromCache(ctx, "sid", "2024", "1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err := repo.GetClassInfosFromCache(ctx, "sid", "2024", "1")
	if err == nil {
		t.Fatalf("expected error after delete")
	}
}

func TestClassInfoCache_Expiration(t *testing.T) {
	repo, mr := newRepoWithLogger(t)
	defer mr.Close()

	ctx := context.Background()
	_ = repo.AddClaInfosToCache(ctx, "sid", "2024", "1", nil) // 黑名单 5 秒

	mr.FastForward(6 * time.Second)

	_, err := repo.GetClassInfosFromCache(ctx, "sid", "2024", "1")
	if err == nil {
		t.Fatalf("expected miss after expiration")
	}
}
