package cache

import (
	"context"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"
)

var (
	ErrKeyNotExists   = redis.Nil
	ErrValeEmptyOrNil = errors.New("缓存值为空或nil")

	DataValueNil   = `null`
	DataValueEmpty = `[]`

	ArchitectureInfosTTL = time.Hour * 24 * 10 // 基本不会变动, 时间可以尽可能的长
	RoomInfosTTL         = time.Hour * 24 * 7
	RoomDetailTTL        = 0 * time.Second // 永不过期
)

const (
	RoomInfosKey         = "ccnubox:elecprice:rooms:%s:%s"
	ArchitectureInfosKey = "ccnubox:elecprice:architecture:%s"
	RoomDetailKey        = "ccnubox:elecprice:detail:%s"
)

type ElecPriceCache interface {
	SetRoomInfos(ctx context.Context, archId, floor, rooms string) error
	GetRoomInfos(ctx context.Context, archId, floor string) (string, error)

	SetArchitectureInfos(ctx context.Context, area, arch string) error
	GetArchitectureInfos(ctx context.Context, area string) (string, error)

	GetRoomDetail(ctx context.Context, roomName string) (string, error)
	SetRoomDetail(ctx context.Context, roomName string, detail string) error
}

type RedisElecPriceCache struct {
	cmd redis.Cmdable
}

func NewRedisElecPriceCache(cmd redis.Cmdable) ElecPriceCache {
	return &RedisElecPriceCache{cmd: cmd}
}

func (cache *RedisElecPriceCache) SetRoomInfos(ctx context.Context, archId, floor, rooms string) error {
	if cache.checkEmptyOrNil(rooms) {
		return ErrValeEmptyOrNil
	}
	key := cache.roomInfosPrefix(archId, floor)

	return cache.cmd.Set(ctx, key, rooms, RoomInfosTTL).Err()
}

func (cache *RedisElecPriceCache) GetRoomInfos(ctx context.Context, archId, floor string) (string, error) {
	key := cache.roomInfosPrefix(archId, floor)

	val, err := cache.cmd.Get(ctx, key).Result()
	if !cache.checkEmptyOrNil(val) && err == nil {
		// 缓存命中且不为空
		return val, nil
	}
	if errors.Is(err, ErrKeyNotExists) {
		// 缓存未命中
		return "", ErrKeyNotExists
	} else {
		return "", err
	}
}

func (cache *RedisElecPriceCache) SetArchitectureInfos(ctx context.Context, area, arch string) error {
	if cache.checkEmptyOrNil(arch) {
		return ErrValeEmptyOrNil
	}
	key := cache.architectureInfosPrefix(area)

	return cache.cmd.Set(ctx, key, arch, ArchitectureInfosTTL).Err()
}

func (cache *RedisElecPriceCache) GetArchitectureInfos(ctx context.Context, area string) (string, error) {
	key := cache.architectureInfosPrefix(area)

	val, err := cache.cmd.Get(ctx, key).Result()
	if !cache.checkEmptyOrNil(val) && err == nil {
		// 缓存命中且不为空
		return val, nil
	}
	if errors.Is(err, ErrKeyNotExists) {
		// 缓存未命中
		return "", ErrKeyNotExists
	} else {
		return "", err
	}
}

func (cache *RedisElecPriceCache) GetRoomDetail(ctx context.Context, roomName string) (string, error) {
	key := cache.roomDetailPrefix(roomName)

	val, err := cache.cmd.Get(ctx, key).Result()
	if !cache.checkEmptyOrNil(val) && err == nil {
		// 缓存命中且不为空
		return val, nil
	}
	if errors.Is(err, ErrKeyNotExists) {
		// 缓存未命中
		return "", ErrKeyNotExists
	} else {
		return "", err
	}
}

func (cache *RedisElecPriceCache) SetRoomDetail(ctx context.Context, roomName string, detail string) error {
	if cache.checkEmptyOrNil(detail) {
		return ErrValeEmptyOrNil
	}
	key := cache.roomDetailPrefix(roomName)

	return cache.cmd.Set(ctx, key, detail, RoomDetailTTL).Err()
}

func (cache *RedisElecPriceCache) roomInfosPrefix(archId, floor string) string {
	return fmt.Sprintf(RoomInfosKey, archId, floor)
}

func (cache *RedisElecPriceCache) roomDetailPrefix(roomName string) string {
	return fmt.Sprintf(RoomDetailKey, roomName)
}

func (cache *RedisElecPriceCache) architectureInfosPrefix(area string) string {
	return fmt.Sprintf(ArchitectureInfosKey, area)
}

func (cache *RedisElecPriceCache) checkEmptyOrNil(value string) bool {
	return value == DataValueNil || value == DataValueEmpty || value == ""
}
