-- 取出待发布的消息
local zsetKey=KEYS[1]
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
            now,
            "LIMIT", 0, 10
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
    local key=id
    local data=redis.call("GET",key)
    if data then
        table.insert(res,data)
        -- 发布成功后由调用方删除，读取时不能提前确认任务。
    elseif isToPublic=="1" then
        -- 清理没有载荷的残留索引，避免占满有限批次使后续任务永远取不到。
        redis.call("ZREM",zsetKey,id)
    end
end
return res