package cache

import (
	"time"

	"github.com/redis/go-redis/v9"
)

type StudentCourseCacheRepo struct {
	rdb            *redis.Client
	metaDataExpire time.Duration
}
