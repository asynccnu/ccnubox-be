package data

import (
	"context"
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

// GetSeatInfos 按楼层查缓存，若缓存不存在或过期会即时更新
func (r *SeatRepo) GetSeatInfos(ctx context.Context, stuID string, roomIDs []string) (map[string][]*biz.Seat, error) {
	now := time.Now()
	result := make(map[string][]*biz.Seat, len(biz.RoomIDs))

	// 循环每个房间
	for _, roomID := range roomIDs {
		// 是否需要后台刷新
		needRefresh := false

		// 情况A：数据全是新鲜的，直接返回
		seats, ts, err := r.cache.GetSeatsByRoom(ctx, roomID)

		if err != nil {
			// 缓存里没有
			r.log.Warnf("get room seats cache(room_id:%s) err: %v", roomID, err)
			needRefresh = true
		} else if ts.IsZero() || now.Sub(*ts) > seatsFreshness {
			// 判断软过期
			needRefresh = true
		}

		// 情况B：房间信息过期，阻塞返回
		if needRefresh {
			key := "lib:getSeatInfos:refresh:" + roomID

			// 为保证可用性，阻塞爬虫座位数据
			_, err, _ := r.sf.Do(key, func() (interface{}, error) {
				refreshCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
				defer cancel()

				return nil, r.SaveRoomSeatsInRedis(refreshCtx, stuID, []string{roomID})
			})

			if err != nil {
				// 如果刷新失败，且之前也没有旧缓存(seats == nil)，那这个房间就真的拿不到数据了
				// 但如果之前是过期数据，seats 还是有的，下面依然可以返回旧数据作为兜底数据
				// 如果这个旧数据太过久远（半小时以上，由硬过期时间配置）可用性太低，直接跳过获取
				r.log.Warnf("refresh room %s failed: %v", roomID, err)
			} else {
				// 刷新成功后，重新从缓存读一次最新数据
				// 因为 SaveRoomSeatsInRedis 只是存到了 Redis，这里要读出来放到 result 里
				// 到这里说明爬虫更新数据成功了，这个房间必有数据
				newSeats, _, err := r.cache.GetSeatsByRoom(ctx, roomID)
				if err == nil {
					seats = newSeats
				}
			}
		}
		result[roomID] = seats
	}

	return result, nil
}
