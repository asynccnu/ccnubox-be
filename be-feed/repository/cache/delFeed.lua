-- 取消发布（删除特定id的消息）
local zsetKey=KEYS[1]
local feedKey=KEYS[2]

redis.call("ZREM",zsetKey,id)
redis.call("DEL",feedKey)

