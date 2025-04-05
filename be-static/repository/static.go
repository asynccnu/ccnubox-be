package repository

import (
	"context"
	"github.com/asynccnu/ccnubox-be/be-static/domain"
	"github.com/asynccnu/ccnubox-be/be-static/pkg/logger"
	"github.com/asynccnu/ccnubox-be/be-static/repository/cache"
	"github.com/asynccnu/ccnubox-be/be-static/repository/dao"
	"github.com/ecodeclub/ekit/slice"
	"time"
)

type StaticRepository interface {
	GetStaticByName(ctx context.Context, name string) (domain.Static, error)
	SaveStatic(ctx context.Context, static domain.Static) error
	GetStaticsByLabels(ctx context.Context, labels map[string]string) ([]domain.Static, error)
}

type CachedStaticRepository struct {
	dao   dao.StaticDAO
	cache cache.StaticCache
	l     logger.Logger
}

func NewCachedStaticRepository(dao dao.StaticDAO, cache cache.StaticCache, l logger.Logger) StaticRepository {
	return &CachedStaticRepository{dao: dao, cache: cache, l: l}
}

func (repo *CachedStaticRepository) GetStaticByName(ctx context.Context, name string) (domain.Static, error) {
	res, err := repo.cache.GetStatic(ctx, name)
	if err == nil {
		return res, nil
	}
	static, err := repo.dao.GetStaticByName(ctx, name)
	res = domain.Static{
		Name:    static.Name,
		Content: static.Content,
		Labels:  static.Labels,
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		er := repo.cache.SetStatic(ctx, res)
		repo.l.Error("回写静态资源失败", logger.Error(er))
	}()
	return res, err
}

func (repo *CachedStaticRepository) SaveStatic(ctx context.Context, static domain.Static) error {
	err := repo.dao.Upsert(ctx, dao.Static{
		Name:    static.Name,
		Content: static.Content,
		Labels:  static.Labels,
	})
	if err != nil {
		return err
	}
	// 更新缓存，或者立刻设置缓存
	return repo.cache.SetStatic(ctx, static)
}

func (repo *CachedStaticRepository) GetStaticsByLabels(ctx context.Context, labels map[string]string) ([]domain.Static, error) {
	statics, err := repo.dao.GetStaticsByLabels(ctx, labels)
	return slice.Map(statics, func(idx int, src dao.Static) domain.Static {
		return domain.Static{
			Name:    src.Name,
			Content: src.Content,
			Labels:  src.Labels,
		}
	}), err
}
