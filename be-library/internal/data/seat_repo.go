package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/asynccnu/ccnubox-be/be-library/internal/biz"
	"github.com/asynccnu/ccnubox-be/be-library/pkg/tool"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/panjf2000/ants/v2"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

const (
	cacheKeyRoomFmt      = "lib:room:%s"
	cacheKeyRoomSeatFmt  = "lib:free_list:seat:%s"
	cacheKeyDataUpdateTs = "lib:room:%s:update_ts"
	// 硬过期，保证夜间不丢缓存
	seatsHardTTL = 24 * time.Hour
	// 软过期，超时则视为需要刷新
	seatsFreshness = 30 * time.Second
)

// 随机预约座位的查询状态
const (
	SEAT = "SEAT" //找到合法座位
	MISS = "MISS" //存在没有缓存的房间
	NONE = "NONE" //没有找到合法座位
)

type SeatRepo struct {
	data    *Data
	sf      singleflight.Group
	crawler biz.LibraryCrawler
	gpool   *ants.Pool
	log     *log.Helper
}

func NewSeatRepo(logger log.Logger, data *Data, crawler biz.LibraryCrawler) (biz.SeatRepo, func()) {
	p, _ := ants.NewPool(30, ants.WithNonblocking(false))
	sr := &SeatRepo{
		data:    data,
		crawler: crawler,
		gpool:   p,
		log:     log.NewHelper(logger),
	}
	return sr, sr.Close
}

func (r *SeatRepo) cacheRoomSeatsKey(roomID string) string {
	return fmt.Sprintf(cacheKeyRoomFmt, roomID)
}

func (r *SeatRepo) cacheRoomSeatsTsKey(seatID string) string {
	return fmt.Sprintf(cacheKeyRoomSeatFmt, seatID)
}

func (r *SeatRepo) cacheRoomUpdateTsKey(roomID string) string {
	return fmt.Sprintf(cacheKeyDataUpdateTs, roomID)
}

func (r *SeatRepo) Close() {
	if r.gpool != nil {
		r.gpool.Release()
		r.log.Info("ClassUsecase goroutine pool released")
	}
}

// 弄个管理员账号来进行持续爬虫
// ZADD seat:{seatID}:times startTimestamp "{start}-{end}"
// HSET roomid timestamp(UnixMilli)
func (r *SeatRepo) SaveRoomSeatsInRedis(ctx context.Context, stuID string, roomID []string) error {
	ttl := r.data.cfg.Redis.Ttl
	// 用pipe收集redis指令，减少网络IO造成的时间损耗
	pipe := r.data.redis.Pipeline()

	allSeats, err := r.crawler.GetSeatInfos(ctx, stuID, roomID)
	if err != nil {
		return err
	}

	ctx2, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	var mu sync.Mutex

	for key, val := range allSeats {
		roomid := key
		seats := val

		for idx, seat := range seats {
			i := idx
			seatInfo := seat
			wg.Add(1)
			submitErr := r.gpool.Submit(func() {
				defer wg.Done()

				freeTime, err := r.crawler.GetFreeList(ctx2, seatInfo.ID, stuID)
				if err != nil {
					r.log.Errorf("crawler get seat:%s freelist error:%v", roomid, err)
					return
				}

				seatInfo.FreeList = freeTime

				mu.Lock()
				seats[i] = seatInfo
				mu.Unlock()
			})

			if submitErr != nil {
				wg.Done()
				r.log.Errorf("submit gpool err:%v", submitErr)
			}

		}
		allSeats[roomid] = seats
	}
	wg.Wait()

	ts := time.Now()

	// 按房间存储 房间里的所有座位数据
	for roomId, seats := range allSeats {
		tskey := r.cacheRoomUpdateTsKey(roomId)
		key := r.cacheRoomSeatsKey(roomId)
		// seatID : seatJson
		hash := make(map[string]string)
		for _, seat := range seats {
			seatID := seat.ID
			seatJson, err := json.Marshal(seat)
			if err != nil {
				r.data.log.Errorf("marshal seat error := %v", err)
				return err
			}
			hash[seatID] = string(seatJson)

			// 建立时间序列 zSet
			zKey := r.cacheRoomSeatsTsKey(seatID)
			var zs []redis.Z
			for _, freeTime := range seat.FreeList {
				startUnix, _ := tool.ParseTodayTimeStringToUnix(freeTime.Start)
				endUnix, _ := tool.ParseTodayTimeStringToUnix(freeTime.End)
				zs = append(zs, redis.Z{
					// 开始时间
					Score: float64(startUnix),
					// 结束时间
					Member: float64(endUnix),
				})
			}
			if len(zs) > 0 {
				pipe.Del(ctx, zKey)
				pipe.ZAdd(ctx, zKey, zs...) // 批量插入时间段
				pipe.Expire(ctx, zKey, ttl.AsDuration())
			}
		}
		// RoomID : {N1111: json1 N2222: json2}
		// 单个房间的座位存储
		pipe.HSet(ctx, key, hash)
		// 房间数据更新时间戳
		pipe.Set(ctx, tskey, ts, 0)
		// 设置 TTL , 过时自动删除捏
		pipe.Expire(ctx, key, ttl.AsDuration())
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		r.data.log.Error("Save SeatInfo in redis ERROR:%s", err.Error())
		return err
	}
	r.data.log.Infof("All seats saved in Redis successfully")
	return nil
}

// GetSeatsByRoomFrom 从缓存获取指定房间的所有座位信息
func (r *SeatRepo) GetSeatsByRoomFromCache(ctx context.Context, roomID string) ([]*biz.Seat, *time.Time, error) {
	roomKey := r.cacheRoomSeatsKey(roomID)
	tsKey := r.cacheRoomUpdateTsKey(roomID)

	data, err := r.data.redis.HGetAll(ctx, roomKey).Result()
	if err != nil {
		r.data.log.Errorf("get seatinfo from redis error (room_id := %s)", roomKey)
		return nil, nil, err
	}
	if len(data) == 0 {
		r.data.log.Errorf("get no seatinfo from redis(room_id := %s)", roomKey)
		return nil, nil, errors.New(fmt.Sprintf("get no seatinfo from redis(room_id := %s)", roomKey))
	}

	tsData, err := r.data.redis.Get(ctx, tsKey).Result()
	if err != nil {
		r.data.log.Errorf("get seatTs from redis error (room_id := %s)", roomKey)
		return nil, nil, err
	}
	if len(tsData) == 0 {
		r.data.log.Errorf("get no seatTs from redis(room_id := %s)", roomKey)
		return nil, nil, errors.New(fmt.Sprintf("get no seatTs from redis(room_id := %s)", roomKey))
	}

	ts, err := time.Parse(time.RFC3339Nano, tsData)
	if err != nil {
		return nil, nil, err
	}

	var seats []*biz.Seat
	for _, v := range data {
		var s biz.Seat
		err = json.Unmarshal([]byte(v), &s)
		if err == nil {
			seats = append(seats, &s)
		}
	}
	return seats, &ts, nil
}

// 返回 座位号 座位是否找到 err
func (r *SeatRepo) FindFirstAvailableSeat(ctx context.Context, start, end int64, roomID []string, stuID string) (string, bool, error) {
	luaScript := `
		local qStart = tonumber(ARGV[1])
		local qEnd = tonumber(ARGV[2])
		
		-- 收集房间ID
		local roomIDs = {}
		for i = 3, #ARGV do
			table.insert(roomIDs, ARGV[i])
		end
		
		-- 获取所有缓存不存在的roomID，如果在缓存存在的room中没找到结果再重新爬取这部分room
		local missRooms={}
		for _, roomID in ipairs(roomIDs) do
			local pattern = "lib:room:" .. roomID
			if redis.call("EXISTS", pattern) == 0 then
				table.insert(missRooms, roomID)
			else
				local seatIDs=redis.call("HKEYS",pattern)
				for _,seatID in ipairs(seatIDs) do
					local zKey="lib:free_list:seat:"..seatID
					local members=redis.call(
							"ZRANGEBYSCORE",
							zKey,
							"-inf",
							qStart,
							"WITHSCORES"
					)
					for j = 1, #members, 2 do
						local endTime = tonumber(members[j])
						if endTime >= qEnd then
							return { "SEAT",seatID }
						end
					end
				end
			end
		end
		
		if #missRooms>0 then
			local result={"MISS"}
			for _,r in ipairs(missRooms) do
				table.insert(result,r)
			end
			return result
		end
		
		return {"NONE"}
	`
	args := make([]interface{}, 0, 2+len(roomID))
	args = append(args, start, end)
	for _, id := range roomID {
		args = append(args, id)
	}

	result, err := r.data.redis.Eval(ctx, luaScript, nil, args...).Result()
	if err != nil {
		r.data.log.Errorf("Error getting first available seat from redis (time:%s)", time.Now().String())
		return "", false, err
	}

	//lua脚本返回的是结构化数组，第一个元素判断查询状态
	resultArr := result.([]interface{})
	switch resultArr[0].(string) {
	case NONE:
		r.data.log.Errorf("No available seat (time:%s)", time.Now().String())
		return "", false, nil
	case SEAT:
		return resultArr[1].(string), true, nil
	case MISS:
		var missRooms []string
		for i := 1; i < len(resultArr); i++ {
			missRooms = append(missRooms, resultArr[i].(string))
		}

		//刷新缓存
		err := r.SaveRoomSeatsInRedis(ctx, stuID, missRooms)
		if err != nil {
			return "", false, err
		}

		return r.FindFirstAvailableSeat(ctx, start, end, missRooms, stuID)
	}

	return "", false, nil
}

// GetSeatInfos 按楼层查缓存
func (r *SeatRepo) GetSeatInfos(ctx context.Context, stuID string, roomIDs []string) (map[string][]*biz.Seat, error) {
	now := time.Now()
	result := make(map[string][]*biz.Seat, len(biz.RoomIDs))
	//用来保存没有缓存的房间，阻塞获取
	missingRooms := make([]string, 0)
	// 循环每个房间
	for _, roomID := range roomIDs {
		//避免go的闭包问题
		roomID := roomID
		// 是否需要后台刷新
		needRefresh := false

		seats, ts, err := r.GetSeatsByRoomFromCache(ctx, roomID)
		if err != nil {
			r.data.log.Warnf("get room seats cache(room_id:%s) err: %v", roomID, err)
			needRefresh = true
		} else if ts.IsZero() || now.Sub(*ts) > seatsFreshness {
			// 判断软过期
			needRefresh = true
		}

		// 这里需要刷新的房间数据不应该是必须得到的吗，这里异步不会导致这几个加载的房间数据无法传递吗
		//如果本来有缓存就异步获取，否则要阻塞
		if needRefresh && len(seats) != 0 {
			go func() {
				bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				_, _, _ = r.sf.Do("lib:getSeatInfos:refresh", func() (interface{}, error) {
					err := r.SaveRoomSeatsInRedis(bgCtx, stuID, []string{roomID})
					return nil, err
				})
			}()

			continue
		}
		if len(seats) == 0 {
			missingRooms = append(missingRooms, roomID)
		} else {
			result[roomID] = seats
		}
	}

	if len(missingRooms) != 0 {
		// 走到这里说明存在完全没有缓存的房间,阻塞一次并拉取座位信息
		// 这里的sf键要与前面的键不同，否则当缓存中完全没有数据时会导致键冲突
		val, err, _ := r.sf.Do("lib:getSeatInfos:refresh:all", func() (interface{}, error) {
			ctx2, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			err := r.SaveRoomSeatsInRedis(ctx2, stuID, missingRooms)
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

	return result, nil
}
