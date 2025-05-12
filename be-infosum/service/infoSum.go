package service

import (
	"context"
	infosumv1 "github.com/asynccnu/ccnubox-be/be-api/gen/proto/infoSum/v1"
	"github.com/asynccnu/ccnubox-be/be-infosum/domain"
	"github.com/asynccnu/ccnubox-be/be-infosum/pkg/errorx"
	"github.com/asynccnu/ccnubox-be/be-infosum/pkg/logger"
	"github.com/asynccnu/ccnubox-be/be-infosum/repository/cache"
	"github.com/asynccnu/ccnubox-be/be-infosum/repository/dao"
	"github.com/asynccnu/ccnubox-be/be-infosum/repository/model"
	"github.com/jinzhu/copier"
	"time"
)

type InfoSumService interface {
	GetInfoSums(ctx context.Context) ([]*domain.InfoSum, error)
	SaveInfoSum(ctx context.Context, req *domain.InfoSum) error
	DelInfoSum(ctx context.Context, id uint) error
}

type infoSumService struct {
	dao   dao.InfoSumDAO
	cache cache.InfoSumCache
	l     logger.Logger
}

// 定义错误函数
var (
	GET_INFOSUM_ERROR = func(err error) error {
		return errorx.New(infosumv1.ErrorGetInfosumError("获取整合信息失败"), "dao", err)
	}

	DEL_INFOSUM_ERROR = func(err error) error {
		return errorx.New(infosumv1.ErrorDelInfosumError("删除整合信息失败"), "dao", err)
	}

	SAVE_INFOSUM_ERROR = func(err error) error {
		return errorx.New(infosumv1.ErrorSaveInfosumError("保存整合信息失败"), "dao", err)
	}
)

func NewInfoSumService(dao dao.InfoSumDAO, cache cache.InfoSumCache, l logger.Logger) InfoSumService {
	return &infoSumService{dao: dao, cache: cache, l: l}
}

func (repo *infoSumService) GetInfoSums(ctx context.Context) ([]*domain.InfoSum, error) {
	//尝试从缓存获取,如果获取直接返回结果
	res, err := repo.cache.GetInfoSums(ctx)
	if err == nil {
		return res, nil
	}

	var resp []*domain.InfoSum
	//如果缓存中不存在则从数据库获取
	webs, err := repo.dao.GetInfoSums(ctx)
	if err != nil {
		return []*domain.InfoSum{}, GET_INFOSUM_ERROR(err)
	}

	//类型转换
	err = copier.Copy(&resp, webs)
	if err != nil {
		return []*domain.InfoSum{}, GET_INFOSUM_ERROR(err)
	}

	go func() {
		err = repo.cache.SetInfoSums(context.Background(), res)
		if err != nil {
			repo.l.Error("回写InfoSums资源失败", logger.FormatLog("cache", err)...)
		}
	}()

	return resp, nil
}

func (repo *infoSumService) SaveInfoSum(ctx context.Context, req *domain.InfoSum) error {
	InfoSum, err := repo.dao.FindInfoSum(ctx, req.ID)
	if InfoSum != nil {
		InfoSum.Name = req.Name
		InfoSum.Link = req.Link
		InfoSum.Description = req.Description
		InfoSum.Image = req.Image
	} else {
		InfoSum = repo.toEntity(req)
	}

	//保存到数据库
	err = repo.dao.SaveInfoSum(ctx, InfoSum)
	if err != nil {
		return SAVE_INFOSUM_ERROR(err)
	}

	//异步写入缓存,牺牲一定的一致性
	go func() {
		ct, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		resp, err := repo.dao.GetInfoSums(ctx)
		if err != nil {
			repo.l.Error("回写InfoSums字段时从dao层中获取失败", logger.FormatLog("cache", err)...)
		}
		var InfoSums []*domain.InfoSum
		err = copier.Copy(resp, InfoSums)
		if err != nil {
			return
		}
		err = repo.cache.SetInfoSums(ct, InfoSums)
		if err != nil {
			repo.l.Error("回写InfoSums资源失败", logger.FormatLog("cache", err)...)
		}
	}()

	return nil
}

func (repo *infoSumService) DelInfoSum(ctx context.Context, id uint) error {
	err := repo.dao.DelInfoSum(ctx, id)
	if err != nil {
		return DEL_INFOSUM_ERROR(err)
	}

	//异步写入缓存,牺牲一定的一致性
	go func() {
		ct, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		resp, err := repo.dao.GetInfoSums(ctx)
		if err != nil {
			repo.l.Error("回写InfoSums字段时从dao层中获取失败", logger.FormatLog("cache", err)...)
		}
		var InfoSums []*domain.InfoSum
		err = copier.Copy(resp, InfoSums)
		if err != nil {
			return
		}
		err = repo.cache.SetInfoSums(ct, InfoSums)
		if err != nil {
			repo.l.Error("回写InfoSums资源失败", logger.FormatLog("cache", err)...)
		}
	}()

	return nil
}

func (repo *infoSumService) toDomain(u *model.InfoSum) *domain.InfoSum {
	return &domain.InfoSum{
		Name:        u.Name,
		Link:        u.Link,
		Description: u.Description,
		Image:       u.Image,
		Model:       u.Model,
	}
}

func (repo *infoSumService) toEntity(u *domain.InfoSum) *model.InfoSum {
	return &model.InfoSum{
		Name:        u.Name,
		Link:        u.Link,
		Description: u.Description,
		Image:       u.Image,
		Model:       u.Model,
	}
}
