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

// 定义 Banner 相关的错误
var (
	GET_BANNERS_ERROR = errorx.FormatErrorFunc(contentv1.ErrorGetBannerError("获取 Banner 列表失败"))

	SAVE_BANNER_ERROR = errorx.FormatErrorFunc(contentv1.ErrorSaveBannerError("保存 Banner 失败"))

	DEL_BANNER_ERROR = errorx.FormatErrorFunc(contentv1.ErrorDelBannerError("删除 Banner 失败"))
)

type BannerService interface {
	GetList(ctx context.Context) ([]domain.Banner, error)
	Save(ctx context.Context, b *domain.Banner) error
	Del(ctx context.Context, id int64) error
}

type bannerService struct {
	repo repository.ContentRepo[model.Banner]
	l    logger.Logger
}

func NewBannerService(repo repository.ContentRepo[model.Banner], l logger.Logger) BannerService {
	return &bannerService{
		repo: repo,
		l:    l,
	}
}

// GetList 获取横幅列表
func (s *bannerService) GetList(ctx context.Context) ([]domain.Banner, error) {
	ms, err := s.repo.GetList(ctx)
	if err != nil {
		// 使用 errorx 记录当前层位置和包装底层错误
		return nil, GET_BANNERS_ERROR(err)
	}
	return s.toDomainList(ms), nil
}

// Save 保存或更新横幅
func (s *bannerService) Save(ctx context.Context, b *domain.Banner) error {
	var m *model.Banner
	var err error

	// 1. 如果有 ID，尝试获取旧数据（Upsert 逻辑的前置检查）
	if b.ID > 0 {
		m, err = s.repo.Get(ctx, "id", b.ID)
		if err != nil && !errors.Is(err, repository.ErrRecordNotFound) {
			return GET_BANNERS_ERROR(errorx.Errorf("保存前查询 Banner(id=%d) 失败: %w", b.ID, err))
		}
	}

	// 2. 如果没找到旧数据或 ID <= 0，初始化新对象
	if m == nil {
		m = &model.Banner{}
	}

	// 字段赋值
	m.WebLink = b.WebLink
	m.PictureLink = b.PictureLink

	// 3. 调用 Repo 保存
	if err := s.repo.Save(ctx, m); err != nil {
		return SAVE_BANNER_ERROR(errorx.Errorf("执行 Banner(id=%d) 保存操作失败: %w", b.ID, err))
	}
	return nil
}

// Del 删除横幅
func (s *bannerService) Del(ctx context.Context, id int64) error {
	// 业务层防御性校验
	if id <= 0 {
		return DEL_BANNER_ERROR(errorx.Errorf("删除 Banner 失败: 传入了无效的 ID (%d)", id))
	}

	if err := s.repo.Del(ctx, "id", id); err != nil {
		return DEL_BANNER_ERROR(errorx.Errorf("删除 Banner(id=%d) 失败: %w", id, err))
	}
	return nil
}

// toDomainList 转换模型逻辑
func (s *bannerService) toDomainList(ms []model.Banner) []domain.Banner {
	res := make([]domain.Banner, 0, len(ms))
	for _, v := range ms {
		res = append(res, domain.Banner{
			ID:          v.ID,
			CreatedAt:   v.CreatedAt,
			UpdatedAt:   v.UpdatedAt,
			WebLink:     v.WebLink,
			PictureLink: v.PictureLink,
		})
	}
	return res
}
