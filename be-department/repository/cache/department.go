package cache

import (
	"context"
	"encoding/json"
	"github.com/asynccnu/ccnubox-be/be-department/domain"
	"github.com/redis/go-redis/v9"
)

type DepartmentCache interface {
	GetDepartments(ctx context.Context) ([]*domain.Department, error)
	SetDepartments(ctx context.Context, departments []*domain.Department) error
}

type RedisDepartmentCache struct {
	cmd redis.Cmdable
}

func NewRedisDepartmentCache(cmd redis.Cmdable) DepartmentCache {
	return &RedisDepartmentCache{cmd: cmd}
}

func (cache *RedisDepartmentCache) GetDepartments(ctx context.Context) ([]*domain.Department, error) {
	key := cache.getKey()
	data, err := cache.cmd.Get(ctx, key).Bytes()
	if err != nil {
		return []*domain.Department{}, err
	}
	var st []*domain.Department
	err = json.Unmarshal(data, &st)
	return st, err
}

func (cache *RedisDepartmentCache) SetDepartments(ctx context.Context, departments []*domain.Department) error {
	key := cache.getKey()
	data, err := json.Marshal(departments)
	if err != nil {
		return err
	}
	return cache.cmd.Set(ctx, key, data, 0).Err() //永不过期
}

func (cache *RedisDepartmentCache) getKey() string {
	return "departments"
}
