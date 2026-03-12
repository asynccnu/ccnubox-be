package data

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/be-classlist/internal/biz"
	"github.com/asynccnu/ccnubox-be/be-classlist/internal/conf"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/redis/go-redis/v9"
)

const (
	recycleIndexKeyTpl = "RecycleIdx:%s:%s:%s"
	recycleValueKeyTpl = "RecycleVal:%s:%s:%s:%s"
)

// RecycleClassInfo 回收站中存储的课程信息结构
type RecycleClassInfo struct {
	Info     ClassInfo     `json:"info"`
	MetaData ClassMetaData `json:"metaData"`
}

var (
	// Lua script for recycling a class
	recycleClassScript = redis.NewScript(`
		local valueKey = KEYS[1]
		local idxKey = KEYS[2]
		local payload = ARGV[1]
		local expireAt = ARGV[2]
		local expiration = ARGV[3]
		local classID = ARGV[4]
		
		redis.call('SET', valueKey, payload, 'EX', expiration)
		redis.call('ZADD', idxKey, expireAt, classID)
		return 1
	`)

	// Lua script for removing a class
	removeClassScript = redis.NewScript(`
		local valueKey = KEYS[1]
		local idxKey = KEYS[2]
		local classID = ARGV[1]
		
		redis.call('ZREM', idxKey, classID)
		redis.call('DEL', valueKey)
		return 1
	`)

	// Lua script for cleaning expired entries
	cleanExpiredScript = redis.NewScript(`
		local idxKey = KEYS[1]
		local now = ARGV[1]
		
		redis.call('ZREMRANGEBYSCORE', idxKey, '-inf', now)
		return 1
	`)
)

type RecycleBinRepo struct {
	rdb               *redis.Client
	recycleExpiration time.Duration
}

func NewRecycleBinRepo(rdb *redis.Client, cf *conf.Server) *RecycleBinRepo {
	expire := 30 * 24 * time.Hour
	if cf.RecycleExpiration > 0 {
		expire = time.Duration(cf.RecycleExpiration) * time.Second
	}
	return &RecycleBinRepo{rdb: rdb, recycleExpiration: expire}
}

func (r RecycleBinRepo) indexKey(stuID, year, semester string) string {
	return fmt.Sprintf(recycleIndexKeyTpl, stuID, year, semester)
}

func (r RecycleBinRepo) valueKey(stuID, year, semester, classID string) string {
	return fmt.Sprintf(recycleValueKeyTpl, stuID, year, semester, classID)
}

func (r RecycleBinRepo) RecycleClass(ctx context.Context, stuID, year, semester, classID string, info *biz.ClassInfoBO) error {
	logh := logger.GetLoggerFromCtx(ctx)
	valueKey := r.valueKey(stuID, year, semester, classID)
	idxKey := r.indexKey(stuID, year, semester)

	// Convert ClassInfoBO to RecycleClassInfo for storage
	recycleInfo := classInfoBOToRecycleClassInfo(info)
	payload, err := json.Marshal(recycleInfo)
	if err != nil {
		return err
	}

	expireAt := time.Now().Add(r.recycleExpiration)

	err = recycleClassScript.Run(ctx, r.rdb,
		[]string{valueKey, idxKey},
		payload,
		expireAt.Unix(),
		int(r.recycleExpiration.Seconds()),
		classID,
	).Err()

	if err != nil {
		logh.Errorf("redis: recycle class failed key=%s class=%s err=%v", idxKey, classID, err)
		return err
	}
	return nil
}

func (r RecycleBinRepo) RemoveClass(ctx context.Context, stuID, year, semester, classID string) error {
	valueKey := r.valueKey(stuID, year, semester, classID)
	idxKey := r.indexKey(stuID, year, semester)

	return removeClassScript.Run(ctx, r.rdb,
		[]string{valueKey, idxKey},
		classID,
	).Err()
}

func (r RecycleBinRepo) GetClass(ctx context.Context, stuID, year, semester, classID string) (*biz.ClassInfoBO, bool) {
	logh := logger.GetLoggerFromCtx(ctx)
	valueKey := r.valueKey(stuID, year, semester, classID)
	idxKey := r.indexKey(stuID, year, semester)

	val, err := r.rdb.Get(ctx, valueKey).Bytes()
	if err == redis.Nil {
		_ = r.rdb.ZRem(ctx, idxKey, classID).Err()
		return nil, false
	}
	if err != nil {
		logh.Errorf("redis: get recycle class key=%s err=%v", valueKey, err)
		return nil, false
	}

	var recycleInfo RecycleClassInfo
	if err := json.Unmarshal(val, &recycleInfo); err != nil {
		logh.Errorf("redis: unmarshal recycle class key=%s err=%v", valueKey, err)
		return nil, false
	}

	// Convert RecycleClassInfo to ClassInfoBO
	classInfoBO := recycleClassInfoToBO(recycleInfo)
	return classInfoBO, true
}

func (r RecycleBinRepo) ListClasses(ctx context.Context, stuID, year, semester string) ([]*biz.ClassInfoBO, error) {
	logh := logger.GetLoggerFromCtx(ctx)
	idxKey := r.indexKey(stuID, year, semester)

	now := time.Now().Unix()
	if err := cleanExpiredScript.Run(ctx, r.rdb, []string{idxKey}, now).Err(); err != nil {
		logh.Warnf("redis: clean expired recycle idx=%s err=%v", idxKey, err)
	}

	ids, err := r.rdb.ZRange(ctx, idxKey, 0, -1).Result()
	if err != nil {
		logh.Errorf("redis: list recycle idx=%s err=%v", idxKey, err)
		return nil, err
	}

	res := make([]*biz.ClassInfoBO, 0, len(ids))
	for _, classID := range ids {
		info, ok := r.GetClass(ctx, stuID, year, semester, classID)
		if !ok {
			continue
		}
		res = append(res, info)
	}
	return res, nil
}

// CleanExpired 删除指定索引下 zset 中已经过期的元素
func (r RecycleBinRepo) CleanExpired(ctx context.Context, stuID, year, semester string) error {
	idxKey := r.indexKey(stuID, year, semester)
	now := time.Now().Unix()
	return cleanExpiredScript.Run(ctx, r.rdb, []string{idxKey}, now).Err()
}

func (r RecycleBinRepo) HasClass(ctx context.Context, stuID, year, semester, classID string) bool {
	idxKey := r.indexKey(stuID, year, semester)
	score, err := r.rdb.ZScore(ctx, idxKey, classID).Result()
	if err != nil {
		return false
	}
	return score > 0
}
