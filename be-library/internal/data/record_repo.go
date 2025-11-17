package data

import (
	"context"

	"github.com/asynccnu/ccnubox-be/be-library/internal/biz"
	"github.com/asynccnu/ccnubox-be/be-library/internal/data/cache"
	"github.com/asynccnu/ccnubox-be/be-library/internal/data/dao"
	"github.com/go-kratos/kratos/v2/log"
)

type recordRepo struct {
	dao   *dao.RecordDAO
	cache *cache.RecordCache
	log   *log.Helper
	cov   *Assembler
}

func NewRecordRepo(dao *dao.RecordDAO, cache *cache.RecordCache, logger log.Logger, cov *Assembler) biz.RecordRepo {
	return &recordRepo{
		dao:   dao,
		cache: cache,
		log:   log.NewHelper(logger),
		cov:   cov,
	}
}

// UpsertFutureRecords 复合唯一键去重,写库成功后删除缓存
func (r *recordRepo) UpsertFutureRecords(ctx context.Context, stuID string, list []*biz.FutureRecords) error {
	dos := ConvertBizFutureRecordsDO(stuID, list)

	if err := r.dao.SyncFutureRecords(ctx, stuID, dos); err != nil {
		return err
	}

	// 写库后删缓存
	if err := r.cache.DelFutureRecords(ctx, stuID); err != nil {
		r.log.Warnf("del future records cache(stu_id:%s) failed: %v", stuID, err)
	}
	return nil
}

// ListFutureRecords 先读缓存,未命中则查库并写回缓存
func (r *recordRepo) ListFutureRecords(ctx context.Context, stuID string) ([]*biz.FutureRecords, error) {
	if cached, ok, err := r.cache.GetFutureRecords(ctx, stuID); err == nil && ok {
		return cached, nil
	} else if err != nil {
		r.log.Warnf("get future records cache(stu_id:%s) err: %v", stuID, err)
	}

	dos, err := r.dao.ListFutureRecords(ctx, stuID)
	if err != nil {
		return nil, err
	}

	out := ConvertDOFutureRecordsBiz(dos)

	// 回填缓存
	if err := r.cache.SetFutureRecords(ctx, stuID, out); err != nil {
		r.log.Warnf("set future records cache(stu_id:%s) err: %v", stuID, err)
	}
	return out, nil
}

func (r *recordRepo) UpsertHistoryRecords(ctx context.Context, stuID string, list []*biz.HistoryRecords) error {
	if len(list) == 0 {
		return nil
	}
	dos := ConvertBizHistoryRecordsDO(stuID, list)

	if err := r.dao.UpsertHistoryRecords(ctx, dos); err != nil {
		return err
	}
	// 写库后删缓存
	if err := r.cache.DelHistoryRecords(ctx, stuID); err != nil {
		r.log.Warnf("del history record cache(stu_id:%s) err: %v", stuID, err)
	}
	return nil
}

// ListHistoryRecords 先读缓存,未命中则查库并写回缓存
func (r *recordRepo) ListHistoryRecords(ctx context.Context, stuID string) ([]*biz.HistoryRecords, error) {
	if cached, ok, err := r.cache.GetHistoryRecords(ctx, stuID); err == nil && ok {
		return cached, nil
	} else if err != nil {
		r.log.Warnf("get history record cache(stu_id:%s) err: %v", stuID, err)
	}

	dos, err := r.dao.ListHistoryRecords(ctx, stuID)
	if err != nil {
		return nil, err
	}

	out := ConvertDOHistoryRecordsBiz(dos)

	// 回填缓存
	if err := r.cache.SetHistoryRecords(ctx, stuID, out); err != nil {
		return nil, err
	}

	return out, nil
}
