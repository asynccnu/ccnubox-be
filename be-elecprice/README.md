# 📘 电费管理服务接口文档

本接口文档基于 `elecprice.v1` 协议，提供电费查询与标准设置等功能的服务定义与错误码说明。

## 🏢 获取建筑信息接口

- **接口名称**：`GetArchitecture`
- **调用方式**：RPC（gRPC）
- **请求路径**：`elecprice.v1.ElecpriceService/GetArchitecture`
- **功能描述**：根据区域名称获取该区域内的建筑物信息。

### ✅ 请求参数（GetArchitectureRequest）

```
{
  "AreaName": "Building Area Name"
}
```

### 📦 响应参数（GetArchitectureResponse）

```
{
  "ArchitectureList": [
    {
      "ArchitectureID": "architecture_1",
      "ArchitectureName": "Building A",
      "BaseFloor": "1",
      "TopFloor": "5"
    },
    {
      "ArchitectureID": "architecture_2",
      "ArchitectureName": "Building B",
      "BaseFloor": "1",
      "TopFloor": "6"
    }
  ]
}
```

### 🚨 可能错误码

| 错误码 | 枚举名         | 描述         |
| ------ | -------------- | ------------ |
| 500    | INTERNAL_ERROR | 系统内部错误 |
| 404    | AREA_NOT_FOUND | 区域未找到   |

## 🏠 获取房间信息接口

- **接口名称**：`GetRoomInfo`
- **调用方式**：RPC（gRPC）
- **请求路径**：`elecprice.v1.ElecpriceService/GetRoomInfo`
- **功能描述**：根据建筑 ID 和楼层获取该楼层内的房间信息。

### ✅ 请求参数（GetRoomInfoRequest）

```
{
  "ArchitectureID": "architecture_1",
  "Floor": "2"
}
```

### 📦 响应参数（GetRoomInfoResponse）

```
{
  "RoomList": [
    {
      "RoomID": "room_101",
      "RoomName": "Room 101"
    },
    {
      "RoomID": "room_102",
      "RoomName": "Room 102"
    }
  ]
}
```

### 🚨 可能错误码

| 错误码 | 枚举名         | 描述         |
| ------ | -------------- | ------------ |
| 500    | INTERNAL_ERROR | 系统内部错误 |
| 404    | ROOM_NOT_FOUND | 房间未找到   |

## 💡 获取电费信息接口

- **接口名称**：`GetPrice`
- **调用方式**：RPC（gRPC）
- **请求路径**：`elecprice.v1.ElecpriceService/GetPrice`
- **功能描述**：根据房间 ID 获取该房间的剩余电费与昨日花费电量信息。

### ✅ 请求参数（GetPriceRequest）

```
{
  "room_id": "room_101"
}
```

### 📦 响应参数（GetPriceResponse）

```
json复制编辑{
  "price": {
    "RemainMoney": "50.00",
    "YesterdayUseValue": "20",
    "YesterdayUseMoney": "10.00"
  }
}
```

### 🚨 可能错误码

| 错误码 | 枚举名         | 描述         |
| ------ | -------------- | ------------ |
| 500    | INTERNAL_ERROR | 系统内部错误 |
| 404    | ROOM_NOT_FOUND | 房间未找到   |

## ⚙️ 设置房间标准接口

- **接口名称**：`SetStandard`
- **调用方式**：RPC（gRPC）
- **请求路径**：`elecprice.v1.ElecpriceService/SetStandard`
- **功能描述**：为指定房间设置标准限制（如电量限制）。

### ✅ 请求参数（SetStandardRequest）

```
{
  "studentId": "2023123456",
  "standard": {
    "limit": 100,
    "room_id": "room_101",
    "room_name": "Room 101"
  }
}
```

### 📦 响应参数（SetStandardResponse）

```
{}
```

### 🚨 可能错误码

| 错误码 | 枚举名          | 描述         |
| ------ | --------------- | ------------ |
| 500    | INTERNAL_ERROR  | 系统内部错误 |
| 400    | INVALID_REQUEST | 请求参数无效 |

## 📜 获取标准列表接口

- **接口名称**：`GetStandardList`
- **调用方式**：RPC（gRPC）
- **请求路径**：`elecprice.v1.ElecpriceService/GetStandardList`
- **功能描述**：获取用户的所有电量标准限制列表。

### ✅ 请求参数（GetStandardListRequest）

```
json复制编辑{
  "studentId": "2023123456"
}
```

### 📦 响应参数（GetStandardListResponse）

```
json复制编辑{
  "standards": [
    {
      "limit": 100,
      "room_id": "room_101",
      "room_name": "Room 101"
    },
    {
      "limit": 150,
      "room_id": "room_102",
      "room_name": "Room 102"
    }
  ]
}
```

### 🚨 可能错误码

| 错误码 | 枚举名             | 描述         |
| ------ | ------------------ | ------------ |
| 500    | INTERNAL_ERROR     | 系统内部错误 |
| 404    | STANDARD_NOT_FOUND | 未找到标准   |

## ⚙️ 取消房间标准接口

- **接口名称**：`CancelStandard`
- **调用方式**：RPC（gRPC）
- **请求路径**：`elecprice.v1.ElecpriceService/CancelStandard`
- **功能描述**：取消某个房间的电量标准限制。

### ✅ 请求参数（CancelStandardRequest）

```
{
  "studentId": "2023123456",
  "room_id": "room_101"
}
```

### 📦 响应参数（CancelStandardResponse）

```
{}
```

### 🚨 可能错误码

| 错误码 | 枚举名         | 描述         |
| ------ | -------------- | ------------ |
| 500    | INTERNAL_ERROR | 系统内部错误 |
| 404    | ROOM_NOT_FOUND | 房间未找到   |

## 🔗 下游依赖服务

1. `be-feed`：发送电费过低消息提醒给指定用户

## 📌 特别说明

1. 爬取的目标网站为:[能源易支付](https://jnb.ccnu.edu.cn/MobileWebPayStandard_Vue/#/addRoom)
2. 本服务依赖外部系统数据，建议调用方做好容错与重试机制，特别是在网络不稳定时。