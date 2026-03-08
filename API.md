# QQ 农场 Bot 外部 API 文档

## 配置

在 `config.json` 中设置 `api_key` 以启用外部 API：

```json
{
  "listen": "0.0.0.0:18080",
  "jwt_secret": "qq-farm-bot-secret-change-me",
  "db_path": "data/farm.db",
  "admin_user": "admin",
  "admin_pass": "admin123",
  "game_server_url": "wss://gate-obt.nqf.qq.com/prod/ws",
  "client_version": "1.6.2.18_20260227",
  "api_key": "your-secret-api-key-here"
}
```

> **`api_key` 为空时外部 API 不启用。** 设置非空值后重启服务即可使用。

---

## 认证方式

**Base URL**: `http://<host>:18080/api/external`

所有接口需要通过以下任一方式传递 API Key：

| 方式 | 示例 |
|------|------|
| **请求头**（推荐） | `X-API-Key: your-secret-api-key-here` |
| **Query 参数** | `?api_key=your-secret-api-key-here` |

认证失败返回 `401`：

```json
{ "error": "invalid or missing API key" }
```

---

## 接口速查表

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/external/accounts` | 获取所有账号列表 |
| `PUT` | `/api/external/accounts/:id/code` | 按 ID 上传登录 code |
| `POST` | `/api/external/code` | 按名称上传 code（不存在则自动创建） |
| `POST` | `/api/external/bot/:id/start` | 启动单个 Bot |
| `POST` | `/api/external/bot/:id/stop` | 停止单个 Bot |
| `POST` | `/api/external/bot/:id/restart` | 重启单个 Bot |
| `POST` | `/api/external/bot/start-all` | 启动所有 Bot |
| `POST` | `/api/external/bot/stop-all` | 停止所有 Bot |
| `GET` | `/api/external/bot/:id/status` | 查询单个 Bot 详细状态 |
| `GET` | `/api/external/status` | 查询全局状态总览 |

---

## 1. 账号管理

### 1.1 获取账号列表

```
GET /api/external/accounts
```

**Response** `200`：

```json
[
  {
    "id": 1,
    "name": "我的QQ农场",
    "platform": "qq",
    "has_code": true,
    "status": "running"
  },
  {
    "id": 2,
    "name": "微信号",
    "platform": "wx",
    "has_code": false,
    "status": "stopped"
  }
]
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | int | 账号 ID |
| `name` | string | 显示名称 |
| `platform` | string | 平台 `qq` / `wx` |
| `has_code` | bool | 是否已设置 login code |
| `status` | string | `running` / `stopped` / `error` |

**curl 示例**：

```bash
curl http://localhost:18080/api/external/accounts \
  -H "X-API-Key: your-secret-api-key-here"
```

---

### 1.2 上传登录 Code（按账号 ID）

```
PUT /api/external/accounts/:id/code
```

**Request Body**：

```json
{
  "code": "abc123def456",
  "platform": "qq"
}
```

| 字段 | 必填 | 说明 |
|------|------|------|
| `code` | ✅ | 登录 code |
| `platform` | ❌ | 覆盖平台（`qq`/`wx`），不传则保持原值 |

**Response** `200`：

```json
{
  "message": "code updated",
  "account_id": 1
}
```

**错误响应**：

| 状态码 | 说明 |
|--------|------|
| `400` | code 为空 |
| `404` | 账号不存在 |

**curl 示例**：

```bash
curl -X PUT http://localhost:18080/api/external/accounts/1/code \
  -H "X-API-Key: your-secret-api-key-here" \
  -H "Content-Type: application/json" \
  -d '{"code": "abc123def456"}'
```

---

### 1.3 上传登录 Code（按名称，自动创建）

```
POST /api/external/code
```

按账号名称匹配：
- **名称已存在** → 更新 code
- **名称不存在** → 自动创建新账号（所有自动化功能默认开启）

**Request Body**：

```json
{
  "name": "我的QQ农场",
  "code": "abc123def456",
  "platform": "qq",
  "auto_start": true
}
```

| 字段 | 必填 | 说明 |
|------|------|------|
| `name` | ✅ | 账号名称 |
| `code` | ✅ | 登录 code |
| `platform` | ❌ | 平台，默认 `qq` |
| `auto_start` | ❌ | 服务启动时自动运行，仅创建新账号时生效 |

**Response（更新已有账号）** `200`：

```json
{
  "message": "code updated",
  "account_id": 1,
  "created": false
}
```

**Response（创建新账号）** `201`：

```json
{
  "message": "account created",
  "account_id": 3,
  "created": true
}
```

**curl 示例**：

```bash
curl -X POST http://localhost:18080/api/external/code \
  -H "X-API-Key: your-secret-api-key-here" \
  -H "Content-Type: application/json" \
  -d '{"name": "小号", "code": "xyz789", "platform": "qq", "auto_start": true}'
```

---

## 2. Bot 控制

### 2.1 启动单个 Bot

```
POST /api/external/bot/:id/start
```

**Response** `200`：

```json
{ "message": "started", "account_id": 1 }
```

**错误响应**：

| 状态码 | 说明 |
|--------|------|
| `400` | 账号未设置 login code |
| `404` | 账号不存在 |
| `500` | 已在运行或启动失败 |

**curl 示例**：

```bash
curl -X POST http://localhost:18080/api/external/bot/1/start \
  -H "X-API-Key: your-secret-api-key-here"
```

---

### 2.2 停止单个 Bot

```
POST /api/external/bot/:id/stop
```

**Response** `200`：

```json
{ "message": "stopped", "account_id": 1 }
```

**错误响应**：

| 状态码 | 说明 |
|--------|------|
| `500` | Bot 未运行或不存在 |

**curl 示例**：

```bash
curl -X POST http://localhost:18080/api/external/bot/1/stop \
  -H "X-API-Key: your-secret-api-key-here"
```

---

### 2.3 重启单个 Bot

```
POST /api/external/bot/:id/restart
```

先停止再启动，中间有 500ms 等待确保清理完成。

**Response** `200`：

```json
{ "message": "restarted", "account_id": 1 }
```

**错误响应**：

| 状态码 | 说明 |
|--------|------|
| `400` | 账号未设置 login code |
| `404` | 账号不存在 |
| `500` | 启动失败 |

**curl 示例**：

```bash
curl -X POST http://localhost:18080/api/external/bot/1/restart \
  -H "X-API-Key: your-secret-api-key-here"
```

---

### 2.4 启动所有 Bot

```
POST /api/external/bot/start-all
```

启动所有已设置 code 的账号，未设置 code 的跳过。

**Response** `200`：

```json
{
  "message": "started 3 bots, 1 failed, 2 skipped (no code)",
  "started": 3,
  "failed": 1,
  "skipped": 2,
  "errors": ["#4(测试号): bot #4 already running"]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `started` | int | 成功启动数量 |
| `failed` | int | 启动失败数量 |
| `skipped` | int | 因无 code 跳过数量 |
| `errors` | string[] | 失败详情 |

**curl 示例**：

```bash
curl -X POST http://localhost:18080/api/external/bot/start-all \
  -H "X-API-Key: your-secret-api-key-here"
```

---

### 2.5 停止所有 Bot

```
POST /api/external/bot/stop-all
```

**Response** `200`：

```json
{
  "message": "stopped 3 bots",
  "stopped": 3
}
```

**curl 示例**：

```bash
curl -X POST http://localhost:18080/api/external/bot/stop-all \
  -H "X-API-Key: your-secret-api-key-here"
```

---

## 3. 状态查询

### 3.1 查询单个 Bot 详细状态

```
GET /api/external/bot/:id/status
```

**Response** `200`：

```json
{
  "account_id": 1,
  "running": true,
  "gid": 123456789,
  "name": "我的农场",
  "level": 25,
  "exp": 12345,
  "gold": 67890,
  "platform": "qq",
  "started_at": "2026-03-08T10:00:00Z",
  "exp_rate_per_hour": 500.5,
  "next_level_exp": 20000,
  "exp_to_next_level": 7655,
  "hours_to_next_level": 15.3,
  "total_harvest": 120,
  "total_steal": 45,
  "total_help": 30,
  "friends_count": 15,
  "total_lands": 18,
  "unlocked_lands": 15,
  "lands": [
    {
      "id": 1,
      "level": 3,
      "max_level": 5,
      "unlocked": true,
      "crop_name": "白萝卜",
      "crop_id": 101,
      "phase": "growing"
    }
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `account_id` | int | 账号 ID |
| `running` | bool | 是否运行中 |
| `gid` | int | 游戏 GID |
| `name` | string | 游戏内昵称 |
| `level` | int | 等级 |
| `exp` | int | 当前经验 |
| `gold` | int | 金币 |
| `platform` | string | 平台 |
| `started_at` | string | 启动时间（ISO 8601） |
| `error` | string | 错误信息（仅异常时返回） |
| `exp_rate_per_hour` | float | 每小时经验速率 |
| `next_level_exp` | int | 下一等级所需总经验 |
| `exp_to_next_level` | int | 距下一等级还差经验 |
| `hours_to_next_level` | float | 预计升级所需小时数 |
| `total_harvest` | int | 总收获次数 |
| `total_steal` | int | 总偷菜次数 |
| `total_help` | int | 总帮忙次数 |
| `friends_count` | int | 好友数量 |
| `total_lands` | int | 总土地数 |
| `unlocked_lands` | int | 已解锁土地数 |
| `lands` | array | 土地详情列表 |

**错误响应**：

| 状态码 | 说明 |
|--------|------|
| `404` | 账号不存在 |

**curl 示例**：

```bash
curl http://localhost:18080/api/external/bot/1/status \
  -H "X-API-Key: your-secret-api-key-here"
```

---

### 3.2 查询全局状态总览

```
GET /api/external/status
```

**Response** `200`：

```json
{
  "total": 5,
  "running": 3,
  "total_gold": 203670,
  "bots": [
    {
      "account_id": 1,
      "name": "主号",
      "platform": "qq",
      "status": "running",
      "level": 25,
      "gold": 67890,
      "exp": 12345
    },
    {
      "account_id": 2,
      "name": "小号",
      "platform": "qq",
      "status": "stopped",
      "level": 10,
      "gold": 0,
      "exp": 0
    },
    {
      "account_id": 3,
      "name": "异常号",
      "platform": "wx",
      "status": "error",
      "level": 0,
      "gold": 0,
      "exp": 0,
      "error": "login failed: invalid code"
    }
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `total` | int | 总账号数 |
| `running` | int | 运行中的 Bot 数 |
| `total_gold` | int | 运行中账号的总金币 |
| `bots` | array | 各 Bot 简要状态 |
| `bots[].account_id` | int | 账号 ID |
| `bots[].name` | string | 账号名称 |
| `bots[].platform` | string | 平台 |
| `bots[].status` | string | `running` / `stopped` / `error` |
| `bots[].level` | int | 等级 |
| `bots[].gold` | int | 金币 |
| `bots[].exp` | int | 经验 |
| `bots[].error` | string | 错误信息（仅异常时） |

**curl 示例**：

```bash
curl http://localhost:18080/api/external/status \
  -H "X-API-Key: your-secret-api-key-here"
```

---

## 4. 典型使用流程

```bash
API_KEY="your-secret-api-key-here"
HOST="http://localhost:18080"

# 1. 上传 code 并自动创建账号
curl -X POST $HOST/api/external/code \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"name": "我的农场", "code": "从抓包获取的code", "platform": "qq"}'

# 2. 启动 bot（假设返回的 account_id 为 1）
curl -X POST $HOST/api/external/bot/1/start \
  -H "X-API-Key: $API_KEY"

# 3. 查看运行状态
curl $HOST/api/external/status \
  -H "X-API-Key: $API_KEY"

# 4. code 过期后重新上传并重启
curl -X PUT $HOST/api/external/accounts/1/code \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"code": "新的code"}'

curl -X POST $HOST/api/external/bot/1/restart \
  -H "X-API-Key: $API_KEY"
```
