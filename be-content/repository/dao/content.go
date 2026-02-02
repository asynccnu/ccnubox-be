package dao

import (
	"context"
	"errors"

	"github.com/asynccnu/ccnubox-be/be-content/repository/model"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"gorm.io/gorm"
)

var (
	ErrRecordNotFound = errorx.New("record not found")
)

type DAO[T model.Content] interface {
	// 上部分是用于对 index 进行处理,下部分是对具体的 feedEvent 进行处理
	FindAll(ctx context.Context) ([]T, error)
	FindOneByField(ctx context.Context, field string, value any) (*T, error)
	Save(ctx context.Context, t *T) error
	DeleteByField(ctx context.Context, field string, value any) error
}

// dao 是一个通用的泛型 dao 实现
type dao[T model.Content] struct {
	db *gorm.DB
}

// NewGormDAO 创建泛型 dao 实例
func NewGormDAO[T model.Content](db *gorm.DB) DAO[T] {
	return &dao[T]{db: db}
}

// FindAll 获取所有记录
func (dao *dao[T]) FindAll(ctx context.Context) ([]T, error) {
	var ts []T
	if err := dao.db.WithContext(ctx).Find(&ts).Error; err != nil {
		return nil, errorx.Errorf("查询所有记录失败: %w", err)
	}
	return ts, nil
}

// FindOneByField 根据指定字段查询单条记录
func (dao *dao[T]) FindOneByField(ctx context.Context, field string, value any) (*T, error) {
	var t T
	if err := dao.db.WithContext(ctx).Where(field+" = ?", value).First(&t).Error; err != nil {
		// 特殊处理：如果记录不存在，通常返回原生错误或特定 nil 状态，方便上层业务判断
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRecordNotFound
		}
		// 其他数据库异常进行包装
		return nil, errorx.Errorf("根据字段查询记录失败: %w, field: %s, value: %v", err, field, value)
	}
	return &t, nil
}

// Save 保存或更新记录
func (dao *dao[T]) Save(ctx context.Context, t *T) error {
	if err := dao.db.WithContext(ctx).Save(t).Error; err != nil {
		return errorx.Errorf("保存记录失败: %w, data: %+v", err, t)
	}
	return nil
}

// DeleteByField 根据字段删除记录
func (dao *dao[T]) DeleteByField(ctx context.Context, field string, value any) error {
	var t T
	if err := dao.db.WithContext(ctx).Where(field+" = ?", value).Delete(&t).Error; err != nil {
		return errorx.Errorf("删除记录失败: %w, field: %s, value: %v", err, field, value)
	}
	return nil
}
