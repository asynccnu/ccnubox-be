-- 取消发布（删除特定id的消息）
local zsetKey=KEYS[1]
local prefix=KEYS[2]
local id=ARGV[1]

local feedKey=prefix..id
redis.call("ZREM",zsetKey,id)
redis.call("DEL",feedKey)

