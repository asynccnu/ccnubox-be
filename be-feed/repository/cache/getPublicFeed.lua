-- 取出待发布的消息
local zsetKey=KEYS[1]
local prefix=KEYS[2]
local now=ARGV[1]

local ids=redis.call(
        "ZRANGEBYSCORE",
        zsetKey,
        0,
        now
)

local res={}

for _,id in ipairs(ids) do
    local key=prefix .. id
    local data=redis.call("GET",key)
    if data then
        table.insert(res,data)
        redis.call("ZREM",zsetKey,id)
        redis.call("DEL",key)
    end
end
return res