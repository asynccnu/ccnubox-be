# 📘 用户 Cookie 获取服务接口文档

本接口文档基于 `ccnu.v1` 协议，提供与华中师范大学教务系统交互的服务定义与错误码说明。

## 🍪 获取用户 Cookie 接口

- **接口名称**：`GetCCNUCookie`
- **调用方式**：RPC（gRPC）
- **请求路径**：`ccnu.v1.CCNUService/GetCCNUCookie`
- **功能描述**：根据学号和密码从教务系统获取认证 Cookie，用于后续模拟登录等操作。

### ✅ 请求参数（GetCCNUCookieRequest）

```
{
  "student_id": "2023123456",
  "password": "your_password"
}
```

### 📦 响应参数（GetCCNUCookieResponse）

```
{
  "cookie": "ccnu_auth_cookie_string"
}
```

### 🚨 可能错误码

| 错误码 | 枚举名             | 描述           |
| ------ | ------------------ | -------------- |
| 401    | INVALID_SID_OR_PWD | 学号或密码错误 |
| 501    | CCNUSERVER_ERROR   | 教务服务器异常 |
| 502    | SYSTEM_ERROR       | 系统内部错误   |

## 🔗 下游依赖服务

1. `be-ccnu`：对接华中师范大学教务系统，实现登录模拟与 Cookie 获取。

## 📌 特别说明

本服务依赖外部教务系统响应，建议调用方做好容错与重试机制。
