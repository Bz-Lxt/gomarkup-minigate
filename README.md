# MiniGate

轻量级 API 网关（Mini Kong / APISIX 风格）。Go 数据面 + Vue 管理台。开发期端口见下文（随机端口）。完整交付文档将在 `/deploy` 阶段按 8081+ 标准端口重写。

## 1. 如何启动

```bash
docker compose up --build -d
```

等待 `gateway` healthy 后访问管理台。

## 2. 使用说明

打开管理台后可 CRUD 路由 / 上游 / 中间件。保存即写入 `config/gateway.yaml` 并热更新，无需重启。数据面试探：

```bash
curl http://127.0.0.1:18480/echo/ping
curl http://127.0.0.1:18480/secure/x   # 401
# 管理台「中间件」页签发 JWT 后再带 Authorization: Bearer <token>
```

## 3. 服务列表及 API 说明

| 服务 | URL |
|---|---|
| Admin UI | http://localhost:18482 |
| Admin API | http://localhost:18481 |
| Gateway | http://localhost:18480 |
| Upstream A/B | http://localhost:18483 / 18484 |

接口契约见 `docs/API.md`。

## 4. 测试账号

管理台无登录（内网运维面）。JWT 演示 secret：`minigate-dev-secret`，可在中间件页一键签发。

## 5. 题目内容

使用 Go 实现轻量级 API 网关：Radix Tree 动态路由、反向代理与负载均衡、JWT/限流/日志中间件流水线、YAML + fsnotify 配置热更新，并提供后台管理页面。

## 6. 项目结构

`backend/` 网关与 Admin API；`frontend-admin/` 管理台；`mock-upstream/` 演示回源；`config/gateway.yaml` 配置 SSOT；`tests/api_smoke.py` 冒烟。

## 7. API 模拟与切换指南

本项目**不调用按量计费外部 API**。`mock-upstream` 是交付内置的演示回源（真实 HTTP 服务，不是假逻辑）。生产环境将 `config/gateway.yaml` 中的 `nodes.target` 改为真实服务地址即可，无需开关变量。中间件「动态加载」实现为编译期注册 + 运行期配置启停（见 Requirements C-1）。etcd/Consul 为 V2，当前 ConfigSource 仅 File。
