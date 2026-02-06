package repository

import (
	"github.com/asynccnu/ccnubox-be/be-content/repository/cache"
	"github.com/asynccnu/ccnubox-be/be-content/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-content/repository/model"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// NewRepo 是通用的构造逻辑，保持私有或作为内部辅助函数
func NewRepo[T model.Content](
	db *gorm.DB,
	cmd redis.Cmdable,
	l logger.Logger,
) ContentRepo[T] {
	c := cache.NewRedisCache[T](cmd)
	d := dao.NewGormDAO[T](db)
	return NewContentRepo[T](d, c, l)
}
func NewBannerRepo(db *gorm.DB, cmd redis.Cmdable, l logger.Logger) ContentRepo[model.Banner] {
	return NewRepo[model.Banner](db, cmd, l)
}

func NewInfoSumRepo(db *gorm.DB, cmd redis.Cmdable, l logger.Logger) ContentRepo[model.InfoSum] {
	return NewRepo[model.InfoSum](db, cmd, l)
}

func NewWebsiteRepo(db *gorm.DB, cmd redis.Cmdable, l logger.Logger) ContentRepo[model.Website] {
	return NewRepo[model.Website](db, cmd, l)
}

func NewDepartmentRepo(db *gorm.DB, cmd redis.Cmdable, l logger.Logger) ContentRepo[model.Department] {
	return NewRepo[model.Department](db, cmd, l)
}

func NewCalendarRepo(db *gorm.DB, cmd redis.Cmdable, l logger.Logger) ContentRepo[model.Calendar] {
	return NewRepo[model.Calendar](db, cmd, l)
}

func NewVersionRepo(db *gorm.DB, cmd redis.Cmdable, l logger.Logger) ContentRepo[model.Version] {
	return NewRepo[model.Version](db, cmd, l)
}

// ProviderSet 现在包含的是具象的函数名
var ProviderSet = wire.NewSet(
	NewBannerRepo,
	NewInfoSumRepo,
	NewWebsiteRepo,
	NewDepartmentRepo,
	NewCalendarRepo,
	NewVersionRepo,
)
