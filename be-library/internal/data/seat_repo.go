package data

import (
	"context"
	"fmt"
	"time"

	"github.com/asynccnu/ccnubox-be/be-library/internal/biz"
	"github.com/asynccnu/ccnubox-be/be-library/internal/conf"
	"github.com/asynccnu/ccnubox-be/be-library/internal/data/cache"
	"github.com/go-kratos/kratos/v2/log"
	"golang.org/x/sync/singleflight"
)

const (
	// 软过期，超时则视为需要刷新
	seatsFreshness = 30 * time.Second
)

type SeatRepo struct {
	cache   *cache.SeatCache
	cfg     *conf.Data
	log     *log.Helper
	cov     *Assembler
	sf      *singleflight.Group
	crawler biz.LibraryCrawler
}

func NewSeatRepo(seatCache *cache.SeatCache, cfg *conf.Data, logger log.Logger, cov *Assembler, sf *singleflight.Group, crawler biz.LibraryCrawler) biz.SeatRepo {
	return &SeatRepo{
		cache:   seatCache,
		cfg:     cfg,
		log:     log.NewHelper(logger),
		cov:     cov,
		sf:      sf,
		crawler: crawler,
	}
}

// SaveRoomSeatsInRedis 保存房间座位信息到 Redis
func (r *SeatRepo) SaveRoomSeatsInRedis(ctx context.Context, stuID string, roomID []string) error {
	allSeats, err := r.crawler.GetSeatInfos(ctx, stuID, roomID)
	if err != nil {
		return err
	}

	ttl := r.cfg.Redis.Ttl.AsDuration()
	return r.cache.SaveRoomSeats(ctx, stuID, roomID, allSeats, ttl)
}

// FindFirstAvailableSeat 返回 座位号 座位是否找到 err
func (r *SeatRepo) FindFirstAvailableSeat(ctx context.Context, start, end int64, roomID []string) (string, bool, error) {
	return r.cache.FindFirstAvailableSeat(ctx, start, end, roomID)
}

// GetSeatInfos 按楼层查缓存
func (r *SeatRepo) GetSeatInfos(ctx context.Context, stuID string, roomIDs []string) (map[string][]*biz.Seat, error) {
	now := time.Now()
	result := make(map[string][]*biz.Seat, len(biz.RoomIDs))

	// 是否需要后台刷新
	needRefresh := false

	// 循环每个房间
	for _, roomID := range roomIDs {
		seats, ts, err := r.cache.GetSeatsByRoom(ctx, roomID)
		if err != nil {
			r.log.Warnf("get room seats cache(room_id:%s) err: %v", roomID, err)
			needRefresh = true
		} else if ts.IsZero() || now.Sub(*ts) > seatsFreshness {
			// 判断软过期
			needRefresh = true
		}

		// 这里需要刷新的房间数据不应该是必须得到的吗，这里异步不会导致这几个加载的房间数据无法传递吗
		if needRefresh {
			go func(roomID string) {
				bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				_, _, _ = r.sf.Do("lib:getSeatInfos:refresh:"+roomID, func() (interface{}, error) {
					err := r.SaveRoomSeatsInRedis(bgCtx, stuID, []string{roomID})
					return nil, err
				})
			}(roomID)

			continue
		}
		result[roomID] = seats
	}

	if len(result) == 0 {
		// 走到这里说明完全没有缓存,阻塞一次并拉取座位信息
		// 这里的sf键要与前面的键不同，否则当缓存中完全没有数据时会导致键冲突
		val, err, _ := r.sf.Do("lib:getSeatInfos:refresh:all", func() (interface{}, error) {
			ctx2, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			err := r.SaveRoomSeatsInRedis(ctx2, stuID, roomIDs)
			if err != nil {
				return nil, err
			}

			return r.GetSeatInfos(ctx2, stuID, roomIDs)

		})
		if err != nil {
			return nil, err
		}
		return val.(map[string][]*biz.Seat), nil

	}
	fmt.Println("repo:", result)

	return result, nil
}
