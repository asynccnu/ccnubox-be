package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/asynccnu/ccnubox-be/be-library/internal/biz"
	"github.com/asynccnu/ccnubox-be/be-library/pkg/tool"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
)

const (
	cacheKeyRoomFmt      = "lib:room:%s"
	cacheKeyRoomSeatFmt  = "lib:room:%s:seat:%s"
	cacheKeyDataUpdateTs = "lib:room:%s:update_ts"
	// 硬过期，保证夜间不丢缓存
	seatsHardTTL = 24 * time.Hour
	// 软过期，超时则视为需要刷新
	seatsCacheFreshness = 30 * time.Second
)

type SeatCache struct {
	redis *redis.Client
	log   *log.Helper
}

func NewSeatCache(redis *redis.Client, logger log.Logger) *SeatCache {
	return &SeatCache{
		redis: redis,
		log:   log.NewHelper(logger),
	}
}

func (c *SeatCache) RoomSeatsKey(roomID string) string {
	return fmt.Sprintf(cacheKeyRoomFmt, roomID)
}

func (c *SeatCache) RoomSeatsTsKey(roomID, seatID string) string {
	return fmt.Sprintf(cacheKeyRoomSeatFmt, roomID, seatID)
}

func (c *SeatCache) RoomUpdateTsKey(roomID string) string {
	return fmt.Sprintf(cacheKeyDataUpdateTs, roomID)
}

// SaveRoomSeats 保存房间座位信息到 Redis
func (c *SeatCache) SaveRoomSeats(ctx context.Context, stuID string, roomID []string, allSeats map[string][]*biz.Seat, ttl time.Duration) error {
	pipe := c.redis.Pipeline()
	ts := time.Now()

	// 按房间存储 房间里的所有座位数据
	for roomId, seats := range allSeats {
		tskey := c.RoomUpdateTsKey(roomId)
		key := c.RoomSeatsKey(roomId)
		// seatID : seatJson
		hash := make(map[string]string)
		for _, seat := range seats {
			seatID := seat.DevID
			seatJson, err := json.Marshal(seat)
			if err != nil {
				c.log.Errorf("marshal seat error := %v", err)
				return err
			}
			hash[seatID] = string(seatJson)

			// 建立时间序列 zSet
			zKey := c.RoomSeatsTsKey(roomId, seatID)
			var zs []redis.Z
			for _, ts := range seat.Ts {
				startUnix, _ := tool.ParseToUnix(ts.Start)
				endUnix, _ := tool.ParseToUnix(ts.End)
				// 记录每个被占用时间的开始与结束的时间戳
				zs = append(zs, redis.Z{
					// 开始时间
					Score: float64(endUnix),
					// 结束时间
					Member: float64(startUnix),
				})
			}
			if len(zs) > 0 {
				pipe.ZAdd(ctx, zKey, zs...) // 批量插入时间段
				pipe.Expire(ctx, zKey, ttl)
			} else if len(zs) == 0 {
				// 给未被占用的座位一个默认值，使得查询脚本能查询到空闲座位
				def := redis.Z{
					Score:  2300,
					Member: 2300,
				}

				pipe.ZAdd(ctx, zKey, def)
				pipe.Expire(ctx, zKey, ttl)
			}

		}
		// RoomID : {N1111: json1 N2222: json2}
		// 单个房间的座位存储
		pipe.HSet(ctx, key, hash)
		// 房间数据更新时间戳
		pipe.Set(ctx, tskey, ts, 0)
		// 设置 TTL , 过时自动删除捏
		pipe.Expire(ctx, key, ttl).Err()
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		c.log.Error("Save SeatInfo in redis ERROR:%s", err.Error())
		return err
	}

	c.log.Infof("seats saved in Redis successfully (room_id:%v)", roomID)
	return nil
}

// GetSeatsByRoom 从缓存获取指定房间的所有座位信息
func (c *SeatCache) GetSeatsByRoom(ctx context.Context, roomID string) ([]*biz.Seat, *time.Time, error) {
	roomKey := c.RoomSeatsKey(roomID)
	tsKey := c.RoomUpdateTsKey(roomID)

	data, err := c.redis.HGetAll(ctx, roomKey).Result()
	if err != nil {
		c.log.Errorf("get seatinfo from redis error (room_id := %s)", roomKey)
		return nil, nil, err
	}
	if len(data) == 0 {
		c.log.Errorf("get no seatinfo from redis(room_id := %s)", roomKey)
		return nil, nil, errors.New(fmt.Sprintf("get no seatinfo from redis(room_id := %s)", roomKey))
	}

	tsData, err := c.redis.Get(ctx, tsKey).Result()
	if err != nil {
		c.log.Errorf("get seatTs from redis error (room_id := %s)", roomKey)
		return nil, nil, err
	}
	if len(data) == 0 {
		c.log.Errorf("get no seatTs from redis(room_id := %s)", roomKey)
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

// FindFirstAvailableSeat 查找第一个可用座位
func (c *SeatCache) FindFirstAvailableSeat(ctx context.Context, start, end int64, roomID []string) (string, bool, error) {
	luaScript := `
		local qStart = tonumber(ARGV[1])
		local qEnd = tonumber(ARGV[2])

		-- 收集房间ID
		local roomIDs = {}
		for i = 3, #ARGV do
			table.insert(roomIDs, ARGV[i])
		end

		-- 遍历所有房间ID
		for _, roomID in ipairs(roomIDs) do
			local cursor = "0"
			repeat
				-- 只扫描当前房间下的 seat
				local pattern = "lib:room:" .. roomID .. ":seat:*"
				local scanResult = redis.call("SCAN", cursor, "MATCH", pattern, "COUNT", 100)
				cursor = scanResult[1]
				local keys = scanResult[2]

				for i = 1, #keys do
					local key = keys[i]
					local members = redis.call("ZRANGE", key, 0, -1, "WITHSCORES")

					local free = true
					for j = 2, #members, 2 do
						local startTime = tonumber(members[j - 1])
						local endTime = tonumber(members[j])
						if startTime < qEnd and endTime > qStart then
							free = false
							break
						end
					end

					if free then
						return key -- 找到空闲座位直接返回
					end
				end
			until cursor == "0"
		end

		return nil
	`
	args := make([]interface{}, 0, 2+len(roomID))
	args = append(args, start, end)
	for _, id := range roomID {
		args = append(args, id)
	}

	result, err := c.redis.Eval(ctx, luaScript, nil, args...).Result()
	// redis.Nil 来做无匹配座位的表示符，返回 false
	if errors.Is(err, redis.Nil) {
		c.log.Errorf("No available seat (time:%s)", time.Now().String())
		return "", false, err
	}
	if err != nil {
		c.log.Errorf("Error getting first available seat from redis (time:%s)", time.Now().String())
		return "", false, err
	}

	resultStr, ok := result.(string)
	if !ok {
		c.log.Errorf("No available seat now (time:%s)", time.Now().String())
		return "", false, fmt.Errorf("no available seat now (time:%s)", time.Now().String())
	}

	idx := strings.LastIndexByte(resultStr, ':')
	freeSeatID := resultStr[idx+1:]

	return freeSeatID, true, nil
}
