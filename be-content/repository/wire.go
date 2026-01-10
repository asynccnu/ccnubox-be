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
func NewRepo[T any](
	db *gorm.DB,
	cmd redis.Cmdable,
	contentType string,
	l logger.Logger,
) ContentRepo[T] {
	c := cache.NewRedisCache[[]T](cmd, contentType)
	d := dao.NewGormDAO[T](db)
	return NewContentRepo[T](d, c, l)
}

func ProvideBannerRepo(db *gorm.DB, cmd redis.Cmdable, l logger.Logger) ContentRepo[model.Banner] {
	return NewRepo[model.Banner](db, cmd, "content", l)
}

func ProvideInfoSumRepo(db *gorm.DB, cmd redis.Cmdable, l logger.Logger) ContentRepo[model.InfoSum] {
	return NewRepo[model.InfoSum](db, cmd, "infosum", l)
}

func ProvideWebsiteRepo(db *gorm.DB, cmd redis.Cmdable, l logger.Logger) ContentRepo[model.Website] {
	return NewRepo[model.Website](db, cmd, "website", l)
}

func ProvideDepartmentRepo(db *gorm.DB, cmd redis.Cmdable, l logger.Logger) ContentRepo[model.Department] {
	return NewRepo[model.Department](db, cmd, "department", l)
}

func ProvideCalendarRepo(db *gorm.DB, cmd redis.Cmdable, l logger.Logger) ContentRepo[model.Calendar] {
	return NewRepo[model.Calendar](db, cmd, "calendar", l)
}

// ProviderSet 现在包含的是具象的函数名
var ProviderSet = wire.NewSet(
	ProvideBannerRepo,
	ProvideInfoSumRepo,
	ProvideWebsiteRepo,
	ProvideDepartmentRepo,
	ProvideCalendarRepo,
)
