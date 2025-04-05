package service

import (
	"context"
	"github.com/asynccnu/ccnubox-be/be-static/domain"
	"github.com/asynccnu/ccnubox-be/be-static/repository"
)

type StaticService interface {
	GetStaticByName(ctx context.Context, name string) (domain.Static, error)
	SaveStatic(ctx context.Context, static domain.Static) error
	GetStaticsByLabels(ctx context.Context, labels map[string]string) ([]domain.Static, error)
}

type staticService struct {
	repo repository.StaticRepository
}

func NewStaticService(repo repository.StaticRepository) StaticService {
	return &staticService{repo: repo}
}

func (s *staticService) GetStaticByName(ctx context.Context, name string) (domain.Static, error) {
	return s.repo.GetStaticByName(ctx, name)
}

func (s *staticService) SaveStatic(ctx context.Context, static domain.Static) error {
	return s.repo.SaveStatic(ctx, static)
}

func (s *staticService) GetStaticsByLabels(ctx context.Context, labels map[string]string) ([]domain.Static, error) {
	return s.repo.GetStaticsByLabels(ctx, labels)
}
