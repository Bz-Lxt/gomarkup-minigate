# MiniGate Admin API

> Base URL：`http://localhost:18481`（开发） / 经 Admin UI 反代 `/api`
> 时区：GMT+8，时间字段格式 `yyyy-MM-dd HH:mm:ss`

## 统一响应

```json
{ "code": 0, "message": "ok", "data": {} }
```

错误：`code != 0`，HTTP 状态与业务码对齐（400 / 404 / 409 / 500）。

| code | HTTP | 含义 |
|---|---|---|
| 0 | 200/201 | 成功 |
| 40001 | 400 | 参数校验失败 |
| 40401 | 404 | 资源不存在 |
| 40901 | 409 | ID 冲突 |
| 50001 | 500 | 内部错误 / 配置落盘失败 |

---

## 端点

### Health

`GET /api/v1/health`

```json
{ "code": 0, "message": "ok", "data": { "status": "up", "time": "2026-08-20 14:00:00" } }
```

### Stats（Dashboard）

`GET /api/v1/stats`

```json
{
  "code": 0,
  "data": {
    "qps": 12.5,
    "total_requests": 1024,
    "active_routes": 4,
    "upstreams": [
      { "id": "echo-rr", "name": "Echo RR", "algorithm": "round_robin", "healthy": 2, "total": 2 }
    ],
    "recent_errors": [
      { "time": "2026-08-20 14:00:00", "message": "upstream timeout" }
    ],
    "hot_reload": {
      "source": "file",
      "last_success": "2026-08-20 14:00:00",
      "last_error": ""
    }
  }
}
```

### Routes

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/v1/routes` | 列表 |
| GET | `/api/v1/routes/:id` | 详情 |
| POST | `/api/v1/routes` | 创建（201） |
| PUT | `/api/v1/routes/:id` | 全量更新 |
| PATCH | `/api/v1/routes/:id/toggle` | 启用/禁用 |
| DELETE | `/api/v1/routes/:id` | 删除 |

请求体：

```json
{
  "id": "echo-route",
  "name": "Echo",
  "path": "/echo/*",
  "methods": ["GET", "POST"],
  "host": "",
  "upstream_id": "echo-rr",
  "middlewares": ["logger"],
  "enabled": true,
  "priority": 10,
  "strip_prefix": "/echo"
}
```

### Upstreams

| 方法 | 路径 |
|---|---|
| GET | `/api/v1/upstreams` |
| GET | `/api/v1/upstreams/:id` |
| POST | `/api/v1/upstreams` |
| PUT | `/api/v1/upstreams/:id` |
| DELETE | `/api/v1/upstreams/:id` |

```json
{
  "id": "echo-rr",
  "name": "Echo RR",
  "algorithm": "round_robin",
  "timeout_ms": 5000,
  "fail_threshold": 3,
  "nodes": [
    { "target": "http://upstream-a:9001", "weight": 1 },
    { "target": "http://upstream-b:9002", "weight": 1 }
  ]
}
```

`algorithm` ∈ `round_robin` | `random` | `weighted_rr` | `least_conn`

可选：`health_path`（默认 `/health`）、`expected_status`（默认 200）、`circuit.enabled` 熔断。

### Middlewares

| 方法 | 路径 |
|---|---|
| GET | `/api/v1/middlewares` |
| PUT | `/api/v1/middlewares/:name` |

```json
{
  "name": "jwt",
  "enabled": true,
  "scope": "global",
  "config": { "secret": "minigate-dev-secret", "header": "Authorization", "skip_paths": ["/health"] }
}
```

内置：`jwt` / `ratelimit` / `logger` / `cors` / `rewrite` / `headers` / `ipfilter`

### Config

`GET /api/v1/config` — 当前生效快照  
`GET /api/v1/config/status` — 热更新状态

### Demo Token

`POST /api/v1/tokens/demo`

```json
{ "code": 0, "data": { "token": "eyJ...", "expires_at": "2026-08-20 15:00:00" } }
```

签发 HS256 JWT，secret 取当前 jwt 中间件配置（缺省 `minigate-dev-secret`），有效期 1 小时。
