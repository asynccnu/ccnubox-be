# 📘 Feed 服务接口文档

该接口文档定义了与 Feed 服务相关的接口，主要用于消息推送管理，包括获取消息、更新消息状态、发布官方消息等功能。

## 🍪 Feed 服务接口

### 1. 获取所有的消息（包括已读和未读）

- **接口名称**：`GetFeedEvents`
- **调用方式**：RPC（gRPC）
- **请求路径**：`feed.v1.FeedService/GetFeedEvents`
- **功能描述**：根据学号获取所有的消息，包括已读和未读消息。

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
      "type": "INFO",
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
      "type": "ALERT",
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

### 4. 更改当前推送的消息数量

- **接口名称**：`ChangeFeedAllowList`
- **调用方式**：RPC（gRPC）
- **请求路径**：`feed.v1.FeedService/ChangeFeedAllowList`
- **功能描述**：更新推送消息的允许列表配置。

#### ✅ 请求参数（ChangeFeedAllowListReq）

```
{
  "allowList": {
    "studentId": "2023123456",
    "grade": true,
    "muxi": true,
    "holiday": false,
    "energy": true
  }
}
```

#### 📦 响应参数（ChangeFeedAllowListResp）

```
{}
```

### 5. 获取 feed 推送许可配置

- **接口名称**：`GetFeedAllowList`
- **调用方式**：RPC（gRPC）
- **请求路径**：`feed.v1.FeedService/GetFeedAllowList`
- **功能描述**：获取当前推送的消息许可配置。

#### ✅ 请求参数（GetFeedAllowListReq）

```
{
  "studentId": "2023123456"
}
```

#### 📦 响应参数（GetFeedAllowListResp）

```
{
  "allowList": {
    "studentId": "2023123456",
    "grade": true,
    "muxi": true,
    "holiday": false,
    "energy": true
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
