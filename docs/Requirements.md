# MiniGate — 轻量级 API 网关需求文档

> 版本：v1.0（冻结） | 日期：2026-08-20 | 时区：GMT+8
> 原始 Prompt：见 `docs/.meta/original_prompt.md`

---

## 1. 项目概述

使用 **Go 语言**实现一个轻量级 API 网关（对标 Mini Kong / APISIX），代号 **MiniGate**。系统由两部分组成：

1. **网关数据面（Gateway）**：高性能 HTTP 反向代理，负责路由匹配、负载均衡、中间件流水线、配置热更新。
2. **后台管理面（Admin UI + Admin API）**：可视化管理路由、上游服务、中间件配置，并实时查看运行状态。

**规模评估**：预估 10,000 – 15,000 LoC（含前端），属于 **10k–40k 区间** → 接受，但 `docs/Roadmap.md` 必须显式划分 MVP / V1 / V2 边界后方可编码。

---

## 2. 功能需求（WHAT）

### FR-1 动态路由（Radix Tree）
- 基于**基数树（Radix Tree）**实现 HTTP 路由匹配（比 Trie 更省内存、缓存友好）。
- 支持：精确路径、前缀匹配（`/api/*`）、路径参数（`/users/{id}`）、Host + Method 维度匹配。
- 路由表线程安全：读写分离（RWMutex 或原子指针 + COW），匹配路径上**不允许出现全局写锁**。
- 路由规则字段：`id, name, path, methods[], host, upstream_id, middlewares[], enabled, priority, created_at, updated_at`。

### FR-2 反向代理与负载均衡
- 基于 `net/http/httputil.ReverseProxy` 实现反向代理，支持 HTTP/1.1，连接池复用。
- 上游（Upstream）模型：一个 Upstream 含多个节点（`host:port` + 权重）。
- 负载均衡算法（按 Upstream 可配）：
  - **轮询（Round Robin）**
  - **随机（Random）**
  - **加权轮询（Weighted RR，平滑算法，参考 Nginx smooth weighted）**
- 被动健康检查：节点连续失败 N 次自动摘除，后台探测恢复（V1 可简化为主动拨测）。
- 超时与重试：可配置代理超时；仅对**幂等请求（GET/HEAD）**做有限重试（最多 1 次，换节点）。

### FR-3 中间件流水线（Plugin 机制）
- 统一 `Middleware` 接口：`Name() string` + `Handle(next http.Handler) http.Handler` + `ValidateConfig(cfg) error`。
- 内置中间件（编译期注册、运行期按配置动态启停/排序）：
  1. **JWT 认证**：HS256 验签、过期校验、可配 `secret / header 名 / 白名单路径`。
  2. **限流**：令牌桶（Token Bucket）与漏桶（Leaky Bucket）两种算法可选；维度支持 全局 / 按路由 / 按客户端 IP。
  3. **访问日志**：统一 Logger（level 可控），记录 方法、路径、状态码、耗时、上游节点、客户端 IP；生产模式屏蔽 debug。
- 中间件可绑定到 **Route 级** 或 **Global 级**，执行顺序可配置。

### FR-4 配置热更新
- 配置文件：YAML（`config/gateway.yaml`），包含路由、上游、中间件全量声明。
- 基于 `fsnotify` 监听文件变化，校验通过后**原子替换**路由表与中间件链，**不重启进程、不中断在途请求**。
- 校验失败：保留旧配置，日志告警 + Admin API 暴露最近一次错误。
- Admin API 写操作 → 持久化到 YAML → 触发同一套热更新流水线（单一事实来源）。
- **etcd/Consul 集成**：V2 预留 `ConfigSource` 接口抽象，MVP 仅实现 File Source。

### FR-5 后台管理页面（Admin UI）
- 技术栈建议：Vue 3 + Element Plus + Vite（最终由架构师定夺）。
- 页面：
  1. **Dashboard**：QPS、活跃路由数、上游节点健康状态、最近错误。
  2. **路由管理**：路由 CRUD、启用/禁用开关、中间件绑定。
  3. **上游管理**：Upstream 与节点 CRUD、权重配置、负载均衡算法选择。
  4. **中间件管理**：全局/路由级中间件配置（表单化编辑 JSON Schema 参数）。
  5. **配置中心**：查看当前生效配置、热更新状态（最后更新时间 / 最近一次错误）。
- 视觉标准：遵循 Redline 2「Aesthetic Excellence」，现代设计系统、合理留白与反馈状态，禁止裸 HTML。

---

## 3. 矛盾检测与技术约束记录

| # | 问题 | 结论 |
|---|------|------|
| C-1 | 需求提到中间件「动态加载」。Go 的 `plugin` 包仅支持 Linux、要求构建环境完全一致、无法卸载，与 Docker 跨平台交付（ARM64+AMD64）冲突。 | **解释为「动态启停 + 动态配置」**：中间件编译期内置注册表，运行期通过热更新配置动态挂载/卸载/调参。这是 Traefik 等纯 Go 网关的通行做法。不交付 `.so` 动态加载。 |
| C-2 | 「etcd/Consul」与「文件监听」是二选一（"或"）。引入 etcd 会增加部署复杂度与规模。 | MVP 只做 **fsnotify 文件监听**；`ConfigSource` 接口抽象，V2 再接 etcd。 |
| C-3 | 开发机为 macOS（darwin），交付须 Docker 化。 | 全部服务容器化，`docker compose up` 一键启动；基础镜像验证 ARM64/AMD64 双架构。 |

---

## 4. 验收基线（可测量）

| 维度 | 指标 |
|------|------|
| 路由匹配性能 | 1,000 条路由规模下，单次匹配 p99 < 1ms（`go test -bench` 基准报告） |
| 代理吞吐 | 本地回源压测（`wrk`/`hey`，回源为 mock 服务）QPS ≥ 10,000，p99 延迟 < 50ms |
| 负载均衡分布 | 1000 次请求下，轮询各节点命中数差 ≤ 1；加权（1:3）分布误差 < 5% |
| 限流精度 | 令牌桶速率 100 QPS 时，60s 窗口放行量误差 < 5% |
| 热更新 | 配置变更到生效 < 2s；在途请求 0 中断；非法配置不生效且有告警 |
| JWT 中间件 | 合法 token 放行、伪造/过期 token 返回 401，白名单路径直通 |
| Admin UI | 5 个页面可用，路由/上游/中间件 CRUD 经 UI 操作后网关行为实时变化 |
| 交付 | `docker compose up --build -d` 一键启动，localhost 可访问 Admin UI 与网关入口 |

---

## 5. 外部依赖评估

- **无计量计费 API**、无实时事实数据依赖 → 不涉及 Mock Legitimacy 问题。
- 后端真实服务：交付物内置 **2 个 mock upstream 服务**（Go 简单 HTTP echo 服务），用于演示负载均衡与代理，属正当演示用途。
- JWT：使用 `github.com/golang-jwt/jwt/v5` 库，非外部服务。

---

## 6. 交付物清单

- `backend/`：网关 + Admin API（Go 单二进制，内嵌或分进程均可，架构师定夺）
- `frontend-admin/`：管理后台（Vue3 构建产物由 Nginx 托管）
- `mock-upstream/`：演示用后端服务 ×2
- `docker-compose.yml`：一键编排（开发期随机端口，交付期 8081+）
- `tests/`：Go 单元测试（Radix Tree / LB / 限流 / JWT）+ API Smoke 测试
- `docs/`：Requirements / Roadmap / DesignSpec / QA_Record / AuditReport / API.md

---

**PM 结论**：✅ 需求清晰、规模可控（10k–15k LoC）、无不可模拟外部依赖、Docker 交付可行。**接受立项**。
