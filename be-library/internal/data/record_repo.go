package data

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/be-library/internal/biz"
	"github.com/asynccnu/ccnubox-be/be-library/internal/data/DO"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm/clause"
)

const (
	RecordKeyPrefix    = "lib:records:"
	RecordTTL          = 60 * time.Second
	RecordUpdatePrefix = "lib:records_update:"
)

type recordRepo struct {
	data *Data
}

func NewRecordRepo(data *Data) biz.RecordRepo {
	return &recordRepo{
		data: data,
	}
}

func (r *recordRepo) recordKey(stuID string) string {
	return fmt.Sprintf("%s%s", RecordKeyPrefix, stuID)
}

func (r *recordRepo) recordUpdateKey(stuID string) string {
	return fmt.Sprintf("%s%s", RecordUpdatePrefix, stuID)
}

// 如果不传date就是查所有预约记录
func (r *recordRepo) getRecordsCache(ctx context.Context, stuID string, date ...time.Time) ([]*biz.Record, bool, error) {
	key := r.recordKey(stuID)
	//先检查key是否存在
	exists, err := r.data.redis.Exists(ctx, key).Result()
	if err != nil || exists == 0 {
		return nil, false, err
	}
	var ops []*redis.ZRangeBy
	if len(date) == 0 {
		ops = append(ops, &redis.ZRangeBy{
			Min: "-inf",
			Max: "+inf",
		})
	} else {
		for _, d := range date {
			start := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, d.Location()).Unix()
			end := time.Date(d.Year(), d.Month(), d.Day(), 23, 59, 59, 0, d.Location()).Unix()
			option := &redis.ZRangeBy{
				Min: fmt.Sprintf("%d", start),
				Max: fmt.Sprintf("%d", end),
			}
			ops = append(ops, option)
		}
	}

	var out []*biz.Record
	pipe := r.data.redis.Pipeline()
	var cmds []*redis.StringSliceCmd
	for _, op := range ops {
		cmd := pipe.ZRangeByScore(ctx, key, op)
		cmds = append(cmds, cmd)
	}
	_, err = pipe.Exec(ctx)
	if err != nil {
		return nil, false, err
	}

	for _, cmd := range cmds {
		vals := cmd.Val()
		for _, v := range vals {
			var rec biz.Record
			if err := json.Unmarshal([]byte(v), &rec); err != nil {
				r.data.log.Errorf("unmarshal record error:%v", err)
				continue
			}
			out = append(out, &rec)
		}
	}
	return out, true, nil
}

func (r *recordRepo) setRecordsCache(ctx context.Context, stuID string, list []*biz.Record) error {
	key := r.recordKey(stuID)
	zset := make([]redis.Z, 0, len(list))
	for _, li := range list {
		data, err := json.Marshal(li)
		if err != nil {
			return err
		}
		zset = append(zset, redis.Z{
			Score:  float64(li.MakeBegin.Unix()),
			Member: data,
		})
	}
	pipe := r.data.redis.Pipeline()
	pipe.Del(ctx, key)
	pipe.ZAdd(ctx, key, zset...)
	pipe.Expire(ctx, key, RecordTTL)
	_, err := pipe.Exec(ctx)
	return err
}

func (r *recordRepo) delRecordsCache(ctx context.Context, stuID string) error {
	key := r.recordKey(stuID)
	return r.data.redis.Del(ctx, key).Err()
}

func (r *recordRepo) setRecordsUpdateTime(ctx context.Context, stuID string) error {
	key := r.recordUpdateKey(stuID)
	t := time.Now().Format("2006-01-02 15:04")
	return r.data.redis.Set(ctx, key, t, 0).Err()
}

// GetRecordUpdateTime 这里的更新时间是数据库数据的更新时间
func (r *recordRepo) GetRecordUpdateTime(ctx context.Context, stuID string) (string, error) {
	key := r.recordUpdateKey(stuID)
	t, err := r.data.redis.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}
	return t, nil
}

// UpsertRecords 复合唯一键去重,写库成功后删除缓存
func (r *recordRepo) UpsertRecords(ctx context.Context, stuID string, list []*biz.Record) error {
	if len(list) == 0 {
		return nil
	}
	dos := ConvertRecordBizToDo(stuID, list)
	if err := r.data.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "id"},
			},
			UpdateAll: true,
		}).
		Create(&dos).Error; err != nil {
		return err
	}
	// 写库后删缓存
	if err := r.delRecordsCache(ctx, stuID); err != nil {
		r.data.log.Warnf("del future records cache(stu_id:%s) failed: %v", stuID, err)
	}
	//更新update的时间
	err := r.setRecordsUpdateTime(ctx, stuID)
	if err != nil {
		return err
	}
	return nil
}

// ListRecords 先读缓存,未命中则查库并写回缓存
func (r *recordRepo) ListRecords(ctx context.Context, stuID string, date ...time.Time) ([]*biz.Record, error) {
	//只根据日期查询
	if cache, ok, err := r.getRecordsCache(ctx, stuID, date...); err == nil && ok {
		return cache, nil
	} else if err != nil {
		r.data.log.Warnf("get records cache(stu_id:%s) err: %v", stuID, err)
	}

	//key失效就全量查询更新
	var dos []*DO.Record
	if err := r.data.db.WithContext(ctx).
		Where("stu_id=?", stuID).
		Order("make_begin DESC").
		Find(&dos).Error; err != nil {
		return nil, err
	}

	out := ConvertRecordDoToBiz(dos)

	//回写缓存
	if err := r.setRecordsCache(ctx, stuID, out); err != nil {
		r.data.log.Warnf("set records cache(stu_id:%s) err: %v", stuID, err)
	}

	//过滤指定日期的记录
	var res []*biz.Record
	for _, rec := range out {
		for _, d := range date {
			if d.Day() == rec.MakeDate.Day() {
				res = append(res, rec)
			}
		}
	}

	return res, nil
}
