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

// InfoSumService 接口化
type InfoSumService interface {
	GetList(ctx context.Context) ([]domain.InfoSum, error)
	Save(ctx context.Context, i *domain.InfoSum) error
	Del(ctx context.Context, id int64) error
}

type infoSumService struct {
	repo repository.ContentRepo[model.InfoSum]
	l    logger.Logger
}

func NewInfoSumService(repo repository.ContentRepo[model.InfoSum], l logger.Logger) InfoSumService {
	return &infoSumService{
		repo: repo,
		l:    l,
	}
}

func (s *infoSumService) GetList(ctx context.Context) ([]domain.InfoSum, error) {
	ms, err := s.repo.GetList(ctx)
	if err != nil {
		return nil, errorx.Errorf("获取信息汇总列表失败: %w", err)
	}
	return s.toDomainList(ms), nil
}

func (s *infoSumService) Save(ctx context.Context, i *domain.InfoSum) error {
	var m *model.InfoSum
	var err error

	// 1. 查找现有数据
	if i.ID > 0 {
		m, err = s.repo.Get(ctx, "id", i.ID)
		if err != nil && !errors.Is(err, repository.ErrRecordNotFound) {
			return errorx.Errorf("保存前按 ID(%d) 查询失败: %w", i.ID, err)
		}
	}

	// 2. 只有在真正没找到时才初始化新对象
	if m == nil {
		m = &model.InfoSum{}
	}

	// 3. 字段赋值
	m.Name = i.Name
	m.Link = i.Link
	m.Description = i.Description
	m.Image = i.Image

	// 4. 执行保存操作并包装错误
	if err := s.repo.Save(ctx, m); err != nil {
		return errorx.Errorf("保存信息汇总(%s)失败: %w", i.Name, err)
	}
	return nil
}

func (s *infoSumService) Del(ctx context.Context, id int64) error {
	if id <= 0 {
		return errorx.Errorf("删除失败: 无效的 ID %d", id)
	}

	if err := s.repo.Del(ctx, "id", id); err != nil {
		return errorx.Errorf("删除信息汇总(id=%d)失败: %w", id, err)
	}
	return nil
}

func (s *infoSumService) toDomainList(ms []model.InfoSum) []domain.InfoSum {
	res := make([]domain.InfoSum, 0, len(ms))
	for _, m := range ms {
		res = append(res, domain.InfoSum{
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
