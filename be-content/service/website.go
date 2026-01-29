package service

import (
	"context"
	"errors"
	"github.com/asynccnu/ccnubox-be/be-content/domain"
	"github.com/asynccnu/ccnubox-be/be-content/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/be-content/repository"
	"github.com/asynccnu/ccnubox-be/be-content/repository/model"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

// WebsiteService 接口化，方便被上层 Handler 调用和 Mock 测试
type WebsiteService interface {
	GetList(ctx context.Context) ([]domain.Website, error)
	Save(ctx context.Context, w *domain.Website) error
	Del(ctx context.Context, id int64) error
}

type websiteService struct {
	repo repository.ContentRepo[model.Website]
	l    logger.Logger
}

func NewWebsiteService(repo repository.ContentRepo[model.Website], l logger.Logger) WebsiteService {
	return &websiteService{
		repo: repo,
		l:    l,
	}
}

func (s *websiteService) GetList(ctx context.Context) ([]domain.Website, error) {
	ms, err := s.repo.GetList(ctx)
	if err != nil {
		// 配合 errorx，提供具体的业务失败描述
		return nil, errorx.Errorf("获取常用网站列表失败: %w", err)
	}
	return s.toDomainList(ms), nil
}

func (s *websiteService) Save(ctx context.Context, w *domain.Website) error {
	var m *model.Website
	var err error

	// 1. 查找现有记录
	if w.ID > 0 {
		m, err = s.repo.Get(ctx, "id", w.ID)
		if err != nil && !errors.Is(err, repository.ErrRecordNotFound) {
			return errorx.Errorf("保存前查询 ID(%d) 失败: %w", w.ID, err)
		}
	} else {
		m, err = s.repo.Get(ctx, "name", w.Name)
		if err != nil && !errors.Is(err, repository.ErrRecordNotFound) {
			return errorx.Errorf("保存前按名称(%s)查询失败: %w", w.Name, err)
		}
	}

	// 2. 只有在确实没找到记录时，才创建新对象
	if m == nil {
		m = &model.Website{}
	}

	// 3. 字段映射
	m.Name = w.Name
	m.Link = w.Link
	m.Description = w.Description
	m.Image = w.Image

	// 4. 保存数据
	if err := s.repo.Save(ctx, m); err != nil {
		return errorx.Errorf("执行网站(%s)保存操作失败: %w", w.Name, err)
	}
	return nil
}

func (s *websiteService) Del(ctx context.Context, id int64) error {
	if id <= 0 {
		return errorx.Errorf("删除失败: 无效的 ID (%d)", id)
	}

	if err := s.repo.Del(ctx, "id", id); err != nil {
		return errorx.Errorf("删除网站(id=%d)失败: %w", id, err)
	}
	return nil
}

func (s *websiteService) toDomainList(ms []model.Website) []domain.Website {
	res := make([]domain.Website, 0, len(ms))
	for _, m := range ms {
		res = append(res, domain.Website{
			ID:          m.ID,
			CreatedAt:   m.CreatedAt,
			UpdatedAt:   m.UpdatedAt,
			Name:        m.Name,
			Link:        m.Link,
			Description: m.Description,
			Image:       m.Image,
		})
	}
	return res
}
