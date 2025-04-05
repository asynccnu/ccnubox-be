# 📘 用户等级管理服务接口文档

本接口文档基于 `counter.v1` 协议，提供对用户查询次数及其等级管理的服务定义与错误码说明。

## 🧮 增加查询次数接口

- **接口名称**：`AddCounter`
- **调用方式**：RPC（gRPC）
- **请求路径**：`counter.v1.CounterService/AddCounter`
- **功能描述**：为指定用户增加一次查询次数，如果该用户不存在则会自动创建。

### ✅ 请求参数（AddCounterReq）

```
{
  "studentId": "2023123456"
}
```

### 📦 响应参数（AddCounterResp）

```
{}
```

### 🚨 可能错误码

| 错误码 | 枚举名          | 描述         |
| ------ | --------------- | ------------ |
| 400    | INVALID_REQUEST | 请求参数无效 |
| 500    | INTERNAL_ERROR  | 系统内部错误 |

## 🧮 获取用户等级列表接口

- **接口名称**：`GetCounterLevels`
- **调用方式**：RPC（gRPC）
- **请求路径**：`counter.v1.CounterService/GetCounterLevels`
- **功能描述**：根据指定的等级标签，获取符合该等级的用户学号列表。

### ✅ 请求参数（GetCounterLevelsReq）

```
{
  "label": "level_1"
}
```

### 📦 响应参数（GetCounterLevelsResp）

```
{
  "studentIds": ["2023123456", "2023123457"]
}
```

### 🚨 可能错误码

| 错误码 | 枚举名          | 描述             |
| ------ | --------------- | ---------------- |
| 404    | LEVEL_NOT_FOUND | 找不到指定的等级 |
| 500    | INTERNAL_ERROR  | 系统内部错误     |

## 🧮 修改用户等级接口

- **接口名称**：`ChangeCounterLevels`
- **调用方式**：RPC（gRPC）
- **请求路径**：`counter.v1.CounterService/ChangeCounterLevels`
- **功能描述**：批量调整用户等级，支持提升或降低。

### ✅ 请求参数（ChangeCounterLevelsReq）

```
{
  "studentIds": ["2023123456", "2023123457"],
  "isReduce": true,
  "step": 1 // 一次调整多少级(0,3,7作为三个等级区间)
}
```

### 📦 响应参数（ChangeCounterLevelsResp）

```
{}
```

### 🚨 可能错误码

| 错误码 | 枚举名          | 描述         |
| ------ | --------------- | ------------ |
| 400    | INVALID_REQUEST | 请求参数无效 |
| 500    | INTERNAL_ERROR  | 系统内部错误 |

## 🧮 清空用户等级接口

- **接口名称**：`ClearCounterLevels`
- **调用方式**：RPC（gRPC）
- **请求路径**：`counter.v1.CounterService/ClearCounterLevels`
- **功能描述**：清空所有用户的等级信息。

### ✅ 请求参数（ClearCounterLevelsReq）

```
{}
```

### 📦 响应参数（ClearCounterLevelsResp）

```
{}
```

### 🚨 可能错误码

| 错误码 | 枚举名         | 描述         |
| ------ | -------------- | ------------ |
| 500    | INTERNAL_ERROR | 系统内部错误 |
