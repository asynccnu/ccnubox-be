-- 取出待发布的消息
local zsetKey=KEYS[1]
local prefix=KEYS[2]
local now=ARGV[1]
local isToPublic=ARGV[2]

local res={}
local ids

-- isToPublic==true:返回的是要发布的消息；isToPublic==false:返回的是全部消息（还未发布）
if isToPublic=="1"
then
    ids=redis.call(
            "ZRANGEBYSCORE",
            zsetKey,
            0,
            now
    )
else
    ids=redis.call(
            "ZRANGEBYSCORE",
            zsetKey,
            0,
            "+inf"
    )
end

for _,id in ipairs(ids) do
    local key=prefix .. id
    local data=redis.call("GET",key)
    if data then
        table.insert(res,data)
        if isToPublic=="1" then
            redis.call("ZREM",zsetKey,id)
            redis.call("DEL",key)
        end
    end
end
return res