package data

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-library/internal/biz"
	"github.com/asynccnu/ccnubox-be/be-library/internal/data/cache"
	"github.com/asynccnu/ccnubox-be/be-library/internal/data/dao"
	"github.com/go-kratos/kratos/v2/log"
)

type creditPointsRepo struct {
	dao   *dao.CreditPointDAO
	cache *cache.CreditPointCache
	log   *log.Helper
	cov   *Assembler
}

func NewCreditPointsRepo(dao *dao.CreditPointDAO, cache *cache.CreditPointCache, logger log.Logger, cov *Assembler) biz.CreditPointsRepo {
	return &creditPointsRepo{
		dao:   dao,
		cache: cache,
		log:   log.NewHelper(logger),
		cov:   cov,
	}
}

// UpsertCreditPoint 复合唯一键去重,写库成功后删除缓存
func (r *creditPointsRepo) UpsertCreditPoint(ctx context.Context, stuID string, list *biz.CreditPoints) error {
	if list == nil {
		return nil
	}

	sum, recs := ConvertBizCreditPointsDO(stuID, list)

	// summary：stu_id 唯一，冲突更新 system/remain/total
	if err := r.dao.UpsertSummary(ctx, sum); err != nil {
		return err
	}

	// records：按 stu_id+title+subtitle+location 去重，冲突忽略
	if err := r.dao.UpsertRecords(ctx, recs); err != nil {
		return err
	}

	// 写库后删缓存
	if err := r.cache.Del(ctx, stuID); err != nil {
		r.log.Warnf("del credit point cache(stu_id:%s) failed: %v", stuID, err)
	}
	return nil
}

func (r *creditPointsRepo) ListCreditPoint(ctx context.Context, stuID string) (*biz.CreditPoints, error) {
	if cached, ok, err := r.cache.Get(ctx, stuID); err == nil && ok {
		return cached, nil
	} else if err != nil {
		r.log.Warnf("get credit point cache(stu_id:%s) err: %v", stuID, err)
	}

	// 读 summary
	sumPtr, err := r.dao.GetSummary(ctx, stuID)
	if err != nil {
		return nil, err
	}

	// 读 records
	recs, err := r.dao.ListRecords(ctx, stuID)
	if err != nil {
		return nil, err
	}

	out := ConvertDOCreditPointsBiz(sumPtr, recs)

	// 回填缓存
	if err := r.cache.Set(ctx, stuID, out); err != nil {
		r.log.Warnf("set credit point cache(stu_id:%s) err: %v", stuID, err)
	}
	return out, nil
}
