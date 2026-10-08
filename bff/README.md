# BFF

基于 Gin 的 BFF。

## 错误码

错误码为五位整数，格式为 `TMMSS`：

- `T` 表示错误责任方：`4` 为客户端错误，`5` 为服务端错误；
- `MM` 表示 BFF 模块；
- `SS` 表示模块内的具体错误场景。

模块 `00` 保留给通用错误。错误码使用显式数值定义，新增错误时不得修改已有错误码。完整定义以 [errs/codes.go](errs/codes.go) 为准。

### 通用错误

| 错误码 | HTTP 状态码 | 含义 |
| :--- | :--- | :--- |
| 40001 | 401 | Authorization 错误 |
| 40002 | 422 | 请求参数错误 |
| 40003 | 403 | 访问权限不足 |
| 40004 | 400 | 非法的参数值 |
| 40005 | 401 | 账号或密码错误 |
| 50001 | 500 | 未分类的系统内部错误 |
| 50002 | 500 | 未预期的错误类型 |
| 50003 | 500 | 类型转换错误 |

### 模块分区

| 模块编号 | 模块 | 服务端错误码范围 | 客户端错误码 |
| :--- | :--- | :--- | :--- |
| 01 | Banner | 50101–50199 | — |
| 02 | Calendar | 50201–50299 | — |
| 03 | InfoSum | 50301–50399 | — |
| 04 | Department | 50401–50499 | — |
| 05 | Card | 50501–50599 | — |
| 06 | Class | 50601–50699 | 40601（课程时间冲突）、40602（课程已存在） |
| 07 | ElecPrice | 50701–50799 | — |
| 08 | Feed | 50801–50899 | — |
| 09 | Question | 50901–50999 | — |
| 10 | Grade | 51001–51099 | — |
| 11 | Static | 51101–51199 | — |
| 12 | User | 51201–51299 | — |
| 13 | Website | 51301–51399 | — |
| 14 | Auth | 51401–51499 | 41401（Authorization 过期） |
| 15 | Library | 51501–51599 | 41514–41518（组队邀请通知） |
| 16 | Swag | 51601–51699 | — |
| 17 | Version | 51701–51799 | — |
| 18 | Semester | 51801–51899 | — |

## 说明

使用长短 Token 机制，请求头使用 Bearer 方式进行身份验证。

## 移动端指标 Collector

`POST /api/v1/metrics/client` 接收移动端批量指标。该路由不依赖用户登录态，使用独立的 App Client Key：

```http
Authorization: Bearer <clientMetrics.clientKey>
Content-Type: application/json
```

```json
{
  "events": [
    {
      "name": "app_error",
      "timestamp": 1740000000000,
      "labels": {
        "platform": "ios",
        "app_version": "1.0.0",
        "module": "course",
        "level": "error"
      },
      "value": 1
    }
  ]
}
```

Collector 仅接受以下固定映射；任何额外 Label 都会导致整批请求被拒绝，避免产生高基数序列：

| 事件名 | Prometheus 指标 | 必需 Label |
| :--- | :--- | :--- |
| `app_error` | `ccnubox_app_error_total` | `platform`, `app_version`, `module`, `level` |
| `app_api_failure` | `ccnubox_app_api_failure_total` | `platform`, `api_group`, `status_code` |
| `app_startup_duration` | `ccnubox_app_startup_duration_seconds` | `platform`, `app_version` |

默认单批最多 100 条、请求体最多 64 KiB、单进程最多接受 32 个不同的 `app_version`。可通过 `clientMetrics` 配置调整。`clientKey` 为空时 Collector 会返回 503。由于客户端内置 Key 只能作为粗粒度防刷措施，公网入口仍应在网关层配置限流。

## 系统架构

// TODO
