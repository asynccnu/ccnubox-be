package service

import (
	"context"
	"errors"

	"github.com/asynccnu/ccnubox-be/be-content/domain"
	"github.com/asynccnu/ccnubox-be/be-content/repository"
	"github.com/asynccnu/ccnubox-be/be-content/repository/model"
	contentv1 "github.com/asynccnu/ccnubox-be/common/api/gen/proto/content/v1"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

// 定义 Department 相关的 RPC 错误
var (
	GET_DEPARTMENTS_ERROR = errorx.FormatErrorFunc(contentv1.ErrorGetDepartmentError("获取部门列表失败"))

	SAVE_DEPARTMENT_ERROR = errorx.FormatErrorFunc(contentv1.ErrorSaveDepartmentError("保存部门信息失败"))

	DEL_DEPARTMENT_ERROR = errorx.FormatErrorFunc(contentv1.ErrorDelDepartmentError("删除部门失败"))
)

type DepartmentService interface {
	GetList(ctx context.Context) ([]domain.Department, error)
	Save(ctx context.Context, d *domain.Department) error
	Del(ctx context.Context, id int64) error
}

type departmentService struct {
	repo repository.ContentRepo[model.Department]
	l    logger.Logger
}

func NewDepartmentService(repo repository.ContentRepo[model.Department], l logger.Logger) DepartmentService {
	return &departmentService{
		repo: repo,
		l:    l,
	}
}

// GetList 获取部门列表
func (s *departmentService) GetList(ctx context.Context) ([]domain.Department, error) {
	ms, err := s.repo.GetList(ctx)
	if err != nil {
		// 使用 GET_DEPARTMENTS_ERROR 包装
		return nil, GET_DEPARTMENTS_ERROR(err)
	}
	return s.toDomainList(ms), nil
}

// Save 保存或更新部门信息
func (s *departmentService) Save(ctx context.Context, d *domain.Department) error {
	var m *model.Department
	var err error

	// 1. 查找现有数据
	if d.ID > 0 {
		m, err = s.repo.Get(ctx, "id", d.ID)
		if err != nil && !errors.Is(err, repository.ErrRecordNotFound) {
			return SAVE_DEPARTMENT_ERROR(errorx.Errorf("保存前按 ID(%d) 查询部门失败: %w", d.ID, err))
		}
	} else {
		// 如果没有 ID，按名称查重
		m, err = s.repo.Get(ctx, "name", d.Name)
		if err != nil && !errors.Is(err, repository.ErrRecordNotFound) {
			return SAVE_DEPARTMENT_ERROR(errorx.Errorf("保存前按名称(%s) 查询部门失败: %w", d.Name, err))
		}
	}

	// 2. 如果不存在则新建
	if m == nil {
		m = &model.Department{}
	}

	// 3. 字段赋值
	m.Name = d.Name
	m.Phone = d.Phone
	m.Place = d.Place
	m.Time = d.Time

	// 4. 执行保存
	if err := s.repo.Save(ctx, m); err != nil {
		return SAVE_DEPARTMENT_ERROR(errorx.Errorf("执行部门(%s)保存操作失败: %w", d.Name, err))
	}
	return nil
}

// Del 删除部门
func (s *departmentService) Del(ctx context.Context, id int64) error {
	if id <= 0 {
		return DEL_DEPARTMENT_ERROR(errorx.Errorf("删除失败: 传入了无效的 ID (%d)", id))
	}

	if err := s.repo.Del(ctx, "id", id); err != nil {
		return DEL_DEPARTMENT_ERROR(errorx.Errorf("删除部门(id=%d)失败: %w", id, err))
	}
	return nil
}

// toDomainList 内部模型转换
func (s *departmentService) toDomainList(ms []model.Department) []domain.Department {
	res := make([]domain.Department, 0, len(ms))
	for _, v := range ms {
		res = append(res, domain.Department{
			ID:        v.ID,
			CreatedAt: v.CreatedAt,
			UpdatedAt: v.UpdatedAt,
			Name:      v.Name,
			Phone:     v.Phone,
			Place:     v.Place,
			Time:      v.Time,
		})
	}
	return res
}
