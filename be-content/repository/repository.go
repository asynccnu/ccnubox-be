package repository

import (
	"context"
	"errors"
	"time"

	"github.com/asynccnu/ccnubox-be/be-content/repository/model"

	"github.com/asynccnu/ccnubox-be/be-content/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/be-content/repository/cache"
	"github.com/asynccnu/ccnubox-be/be-content/repository/dao"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

var (
	ErrRecordNotFound = errors.New("record not found")
)

// ContentRepo 泛型接口
type ContentRepo[T model.Content] interface {
	Get(ctx context.Context, field string, val any) (*T, error)
	Save(ctx context.Context, item *T) error
	Del(ctx context.Context, field string, val any) error
	GetList(ctx context.Context) ([]T, error)
}

type Repository[T model.Content] struct {
	dao   dao.DAO[T]     // 修改为指针
	cache cache.Cache[T] // 列表缓存
	l     logger.Logger
}

func NewContentRepo[T model.Content](dao dao.DAO[T], cache cache.Cache[T], l logger.Logger) ContentRepo[T] {
	return &Repository[T]{
		dao:   dao,
		cache: cache,
		l:     l,
	}
}

// Get 获取单条数据
func (r *Repository[T]) Get(ctx context.Context, field string, val any) (*T, error) {
	res, err := r.dao.FindOneByField(ctx, field, val)
	if err != nil {
		if errors.Is(err, dao.ErrRecordNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, errorx.Errorf("repository获取详情失败: %w, field: %s, val: %v", err, field, val)
	}
	return res, nil
}

// GetList 获取列表（带缓存逻辑）
func (r *Repository[T]) GetList(ctx context.Context) ([]T, error) {
	// 1. 尝试从缓存获取列表
	res, err := r.cache.GetContent(ctx)
	if err == nil {
		return res, nil
	}

	r.l.Info("获取缓存数据失败")

	// 2. 缓存失效，查库
	items, err := r.dao.FindAll(ctx)
	if err != nil {
		return nil, errorx.Errorf("repository获取列表失败: %w", err)
	}

	// 3. 异步回写缓存
	go func() {
		asyncCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		if err := r.cache.SetContent(asyncCtx, items, 0); err != nil {
			r.l.Error("异步回写列表缓存失败", logger.Error(err))
		}
	}()

	return items, nil
}

// Save 保存并更新缓存
func (r *Repository[T]) Save(ctx context.Context, item *T) error {
	err := r.dao.Save(ctx, item)
	if err != nil {
		return errorx.Errorf("repository保存数据失败: %w", err)
	}

	if err := r.cache.ClearContent(ctx); err != nil {
		return errorx.Errorf("删除缓存失败:%w", err)
	}
	return nil
}

// Del 删除并更新缓存
func (r *Repository[T]) Del(ctx context.Context, field string, val any) error {
	err := r.dao.DeleteByField(ctx, field, val)
	if err != nil {
		return errorx.Errorf("repository删除数据失败: %w, field: %s, val: %v", err, field, val)
	}

	if err := r.cache.ClearContent(ctx); err != nil {
		return errorx.Errorf("删除缓存失败:%w", err)
	}
	return nil
}
