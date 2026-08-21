# MiniGate Roadmap

> 版本：v1.0 | 日期：2026-08-20 | 时区：GMT+8
> 规模：10k–15k LoC → 必须显式划分 MVP / V1 / V2

---

## 阶段顺序决策

**UI-First（默认）**。管理后台是标准 CRUD 仪表盘，组件结构由需求字段（路由 / 上游 / 中间件）直接推导，不依赖运行时数据模型生成 UI。先冻结 DesignSpec 与页面骨架，再对接 Admin API。

`frontend-mp` / `frontend-user`：本项目无小程序与 C 端，不创建空壳应用。

---

## 技术栈冻结

| 层 | 选择 |
|---|---|
| 数据面 / 管理 API | Go 1.23，单二进制双端口（Gateway + Admin） |
| 路由引擎 | 自研 Radix Tree（`internal/router`） |
| 反代 | `net/http/httputil.ReverseProxy` |
| 配置 | YAML + `fsnotify`；`ConfigSource` 接口预留 |
| 前端 | Vue 3 + Vite + Vue Router + 自研设计系统 |
| 编排 | Docker Compose，开发期随机端口 |
| 时区 | Asia/Shanghai (GMT+8) |

---

## 端口规划（开发期随机端口）

| 服务 | 容器内 | 宿主机 |
|---|---|---|
| Gateway 数据面 | 8080 | **18480** |
| Admin API | 8081 | **18481** |
| Admin UI | 80 | **18482** |
| Mock Upstream A | 9001 | **18483** |
| Mock Upstream B | 9002 | **18484** |

---

## MVP（本次交付，必须完成）

| ID | 任务 | 状态 |
|---|---|---|
| A-01 | Git 初始化 + `.gitignore` + 目录骨架 | [x] |
| A-02 | `docker-compose.yml` 随机端口编排 | [x] |
| U-01 | `docs/DesignSpec.md` 设计规范 | [x] |
| U-02 | Admin UI：Dashboard / Routes / Upstreams / Middlewares / Config | [x] |
| L-01 | Radix Tree 路由引擎 + 线程安全路由表 | [x] |
| L-02 | 反向代理 + RR / Random / WeightedRR | [x] |
| L-03 | 中间件：JWT / TokenBucket / LeakyBucket / AccessLog | [x] |
| L-04 | YAML 配置 + fsnotify 热更新 + ConfigSource 抽象 | [x] |
| L-05 | Admin REST API（路由/上游/中间件 CRUD） | [x] |
| L-06 | 被动健康检查 + GET/HEAD 单次换节点重试 | [x] |
| L-07 | 2 个 mock upstream + 种子配置 | [x] |
| L-08 | Dockerfile 多阶段 / 双架构基础镜像 | [x] |
| Q-01 | Go 单测（radix / balancer / jwt / ratelimit / config） | [x] |
| Q-02 | API Smoke（health / CRUD / 代理 / JWT / 热更新） | [x] |

## V1（已补齐核心项）

| ID | 任务 | 状态 |
|---|---|---|
| V1-01 | 主动健康检查：自定义 `health_path` / `expected_status` | [x] |
| V1-02 | 熔断器（闭/开/半开）+ Admin stats 暴露 | [x] |
| V1-03 | 路径改写 / 头注入 / CORS / IP 过滤中间件 | [x] |
| V1-04 | 最少连接负载均衡 `least_conn` | [x] |
| V1-05 | TLS 终结、Admin 登录鉴权 | [ ] 仍属后续 |

## V2（远期）

- etcd / Consul `ConfigSource` 实现
- gRPC 代理、Redis 分布式限流、WASM 插件沙箱

---

## 目录结构

```
MiniGate/
├── backend/                 # Go 网关 + Admin API
├── frontend-admin/          # Vue3 管理后台
├── mock-upstream/           # 演示回源服务
├── config/gateway.yaml      # SSOT 配置
├── tests/                   # API Smoke
├── docs/
└── docker-compose.yml
```
