# 📘 Feed 服务接口文档

该接口文档定义了与 Feed 服务相关的接口，主要用于消息推送管理，包括获取消息、更新消息状态、发布官方消息等功能。

## 🍪 Feed 服务接口

### 1. 获取所有的消息（包括已读和未读）

- **接口名称**：`GetFeedEvents`
- **调用方式**：RPC（gRPC）
- **请求路径**：`feed.v1.FeedService/GetFeedEvents`
- **功能描述**：根据学号及当前订阅过滤未删除的消息，再按 `created_at DESC, id DESC` 返回最新 20 条（包括已读和未读）。

#### ✅ 请求参数（GetFeedEventsReq）

```
{
  "studentId": "2023123456"
}
```

#### 📦 响应参数（GetFeedEventsResp）

```
{
  "feedEvents": [
    {
      "id": 1,
      "type": "grade",
      "title": "课程通知",
      "content": "请注意课程安排变动。",
      "read": false,
      "extendFields": {
        "priority": "high"
      },
      "created_at": 1633036800
    },
    {
      "id": 2,
      "type": "energy",
      "title": "电费提醒",
      "content": "你的电费余额即将耗尽。",
      "read": true,
      "extendFields": {
        "alertLevel": "medium"
      },
      "created_at": 1633123200
    }
  ]
}
```

### 2. 更新信息的已读取状态

- **接口名称**：`ReadFeedEvent`
- **调用方式**：RPC（gRPC）
- **请求路径**：`feed.v1.FeedService/ReadFeedEvent`
- **功能描述**：用于更新指定消息的已读取状态。

#### ✅ 请求参数（ReadFeedEventReq）

```
{
  "feedId": 1
}
```

#### 📦 响应参数（ReadFeedEventResp）

```
{}
```

### 3. 清除当前的消息（包括已读和未读）

- **接口名称**：`ClearFeedEvent`
- **调用方式**：RPC（gRPC）
- **请求路径**：`feed.v1.FeedService/ClearFeedEvent`
- **功能描述**：清除指定的消息记录，包括已读和未读消息。

#### ✅ 请求参数（ClearFeedEventReq）

```
{
  "studentId": "2023123456",
  "feedId": 1,
  "status": "read"
}
```

#### 📦 响应参数（ClearFeedEventResp）

```
{}
```

### 4. 更改站内可见性及系统推送订阅

- **接口名称**：`ChangeFeedAllowList`
- **调用方式**：RPC（gRPC）
- **请求路径**：`feed.v1.FeedService/ChangeFeedAllowList`
- **功能描述**：更新六类消息的订阅配置，同时控制站内可见性和系统推送，反馈消息无例外。`library` 未传入时保留原状态。

#### ✅ 请求参数（ChangeFeedAllowListReq）

```
{
  "allowList": {
    "studentId": "2023123456",
    "grade": true,
    "muxi": true,
    "holiday": false,
    "energy": true,
    "feedBack": true,
    "library": true
  }
}
```

#### 📦 响应参数（ChangeFeedAllowListResp）

```
{}
```

### 5. 获取六类消息订阅配置

- **接口名称**：`FindOrCreateAllowList`
- **调用方式**：RPC（gRPC）
- **请求路径**：`feed.v1.FeedService/FindOrCreateAllowList`
- **功能描述**：获取当前六类消息订阅状态，BFF 对应接口为 `GET /feed/getFeedAllowList`。

#### ✅ 请求参数（FindOrCreateAllowListReq）

```
{
  "studentId": "2023123456"
}
```

#### 📦 响应参数（FindOrCreateAllowListResp）

```
{
  "allowList": {
    "studentId": "2023123456",
    "grade": true,
    "muxi": true,
    "holiday": false,
    "energy": true,
    "feedBack": true,
    "library": true
  }
}
```

### 6. 存储用户的 token

- **接口名称**：`SaveFeedToken`
- **调用方式**：RPC（gRPC）
- **请求路径**：`feed.v1.FeedService/SaveFeedToken`
- **功能描述**：保存用户的 token。

#### ✅ 请求参数（SaveFeedTokenReq）

```
{
  "studentId": "2023123456",
  "token": "user_token"
}
```

#### 📦 响应参数（SaveFeedTokenResp）

```
{}
```

### 7. 清除当前账号的 token

- **接口名称**：`RemoveFeedToken`
- **调用方式**：RPC（gRPC）
- **请求路径**：`feed.v1.FeedService/RemoveFeedToken`
- **功能描述**：清除指定账号的 token。

#### ✅ 请求参数（RemoveFeedTokenReq）

```
{
  "studentId": "2023123456",
  "token": "user_token"
}
```

#### 📦 响应参数（RemoveFeedTokenResp）

```
{}
```

### 8. 发布木犀官方消息

- **接口名称**：`PublicMuxiOfficialMSG`
- **调用方式**：RPC（gRPC）
- **请求路径**：`feed.v1.FeedService/PublicMuxiOfficialMSG`
- **功能描述**：发布木犀官方消息。

#### ✅ 请求参数（PublicMuxiOfficialMSGReq）

```
{
  "muxiOfficialMSG": {
    "title": "系统维护通知",
    "content": "我们的系统将在今晚 12:00 进行维护。",
    "extendFields": {
      "priority": "high"
    },
    "publicTime": 1633036800,
    "id": "12345"
  }
}
```

#### 📦 响应参数（PublicMuxiOfficialMSGResp）

```
{}
```

### 9. 停止发布木犀官方消息

- **接口名称**：`StopMuxiOfficialMSG`
- **调用方式**：RPC（gRPC）
- **请求路径**：`feed.v1.FeedService/StopMuxiOfficialMSG`
- **功能描述**：停止发布指定的木犀官方消息。

#### ✅ 请求参数（StopMuxiOfficialMSGReq）

```
{
  "id": "12345"
}
```

#### 📦 响应参数（StopMuxiOfficialMSGResp）

```
{}
```

### 10. 获取待发布的木犀官方消息

- **接口名称**：`GetToBePublicOfficialMSG`
- **调用方式**：RPC（gRPC）
- **请求路径**：`feed.v1.FeedService/GetToBePublicOfficialMSG`
- **功能描述**：获取当前未发布的木犀官方消息列表。

#### ✅ 请求参数（GetToBePublicOfficialMSGReq）

```
{}
```

#### 📦 响应参数（GetToBePublicOfficialMSGResp）

```
{
  "msgList": [
    {
      "title": "系统维护通知",
      "content": "我们的系统将在今晚 12:00 进行维护。",
      "extendFields": {
        "priority": "high"
      },
      "publicTime": 1633036800,
      "id": "12345"
    }
  ]
}
```


## 订阅行为与客户端接入

- 六类消息统一使用小写类型：`energy`、`grade`、`holiday`、`muxi`、`feedback`、`library`。历史大小写差异读取兼容，未知类型拒绝展示和推送。站内列表、gRPC 响应和下发给极光的 `content_type` 均取该小写形式。
- 关闭后隐藏该类型，取消对应的 `pending` 推送，但不删除消息、不清空去重键。重新开启可查看历史消息，已经 `suppressed` 的投递不恢复、不补发。
- 普通消息在关闭期间仍可入库；图书馆保留现有入库前订阅检查。不承诺关闭期间的消息永不入库或永不可见。
- `sending` 不被设置事务抢改，worker 在保存 CID 后读取最新订阅再发送；读取失败走原有重试。最终检查与外部 HTTP 调用间仍有竞争窗口，已提交极光的在途通知可能延迟到达，不保证撤回。
- 位图顺序不变：电费、成绩、节假日、木犀、反馈、图书馆分别为第 0～5 位。显式 `0` 全关闭，`48` 仅反馈和图书馆开启；软删除全关闭。配置缺失时普通类型默认开启，图书馆关闭。设置查询沿用原有创建默认配置的行为；已软删除配置在查询时按全关返回、不自动重建，用户重新保存设置时从有效全关状态恢复该行，未传入的 `library` 保持关闭，显式重新开启会记录新的偏好版本。
- 历史 `feed_fail_events` 仍会由读取接口触发重推，查询和消费软删除均按订阅过滤，不能绕过主列表权限。消费仅限本次读取的记录 ID 及当前学号，避免并发开启订阅或新增记录时误删未读取的消息；未允许的旧失败记录保留，不重构旧存取机制。

### HTTP 字段及全部关闭

现有 HTTP 接口保持不变：`POST /feed/changeFeedAllowList`、`GET /feed/getFeedAllowList`。
HTTP 的反馈字段是 **`feedback`**，不是 `feed_back`；gRPC 原字段为 `feedBack`，服务内部 JSON 为 `feed_back`。客户端必须使用实际 HTTP 定义。
前五项必须传入布尔值，`library` 可选、缺省不修改；“全部关闭”必须显式提交六项，不能只判断其中四项：

```json
{
  "grade": false,
  "muxi": false,
  "holiday": false,
  "energy": false,
  "feedback": false,
  "library": false
}
```

当前仓库不含客户端设置页源码，以下交互需要在客户端仓库实现并验收：

1. 修改开关后串行保存，保存期间禁用重复提交或排队，避免旧请求覆盖新请求。
2. 保存成功后回读 `getFeedAllowList`，用六项服务端状态更新页面，再刷新消息列表；不能仅修改本地缓存就提示已关闭。
3. 保存失败恢复原显示并提示失败；保存成功但回读失败应提示“保存成功，刷新失败”，不能假装已确认服务端状态。
4. 切换账号后清理旧账号状态，重新查询订阅和消息列表；忽略旧账号尚未完成的响应。
5. 页面说明关闭后隐藏、重新开启可查看历史、已取消推送不补发及在途通知无法保证撤回的边界。
