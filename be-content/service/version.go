package service

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-content/repository"
	"github.com/asynccnu/ccnubox-be/be-content/repository/model"
	"github.com/asynccnu/ccnubox-be/common/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/common/pkg/logger"
)

type VersionService interface {
	Get(ctx context.Context) string
	Save(ctx context.Context, version string) error
}

type versionService struct {
	repo    repository.ContentRepo[model.Version]
	version string
	l       logger.Logger
}

func NewVersionService(repo repository.ContentRepo[model.Version], l logger.Logger) VersionService {
	return &versionService{
		repo: repo,
		l:    l,
	}
}

func (s *versionService) Get(ctx context.Context) string {
	ms, err := s.repo.GetList(ctx)
	if err != nil || len(ms) == 0 {
		return s.version
	}
	return ms[0].Version
}

func (s *versionService) Save(ctx context.Context, version string) error {
	ms, err := s.repo.GetList(ctx)
	var m *model.Version
	if err == nil && len(ms) > 0 {
		m = &ms[0]
	} else {
		m = &model.Version{}
	}
	m.Version = version
	if err := s.repo.Save(ctx, m); err != nil {
		return errorx.Errorf("保存版本失败: %w", err)
	}
	return nil
}
