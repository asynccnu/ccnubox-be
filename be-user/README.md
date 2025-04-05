# be-user

## 📘 用户服务接口文档

------

### 🍪 获取用户 Cookie 接口

- **接口名称**：`GetCookie`
- **调用方式**：RPC
- **请求路径**：`UserService/GetCookie`
- **功能描述**：根据学号获取对应的认证 Cookie（如用于模拟教务系统登录）

#### ✅ 请求参数（GetCookieRequest）

```
{
  "studentId": "2023123456"
}
```

#### 📦 响应参数（GetCookieResponse）

```
{
  "cookie": "ccnu_auth_cookie_string"
}
```

#### 🚨 可能错误码

| 错误码 | 描述             |
| ------ | ---------------- |
| 503    | 获取 Cookie 失败 |
| 404    | 用户不存在       |
| 505    | 密码解密失败     |

------

## 🧾 常见错误码说明

| 错误码 | 枚举名称             | 中文描述          |
| ------ | -------------------- | ----------------- |
| 404    | USER_NOT_FOUND_ERROR | 无法找到该用户    |
| 501    | DEFAULT_DAO_ERROR    | 数据库异常        |
| 502    | SAVE_USER_ERROR      | 保存用户失败      |
| 503    | CCNU_GETCOOKIE_ERROR | 获取 Cookie 失败  |
| 504    | ENCRYPT_ERROR        | Password 加密失败 |
| 505    | DECRYPT_ERROR        | Password 解密失败 |

------

## 🔗 涉及下游调用服务

- `be-ccnu`

------

## 🧙 特别说明

> 本服务富含魔法，**如果无法 build 属于正常现象**，请大胆去阅读源码，解锁背后的“谜语人”之术 ✨
