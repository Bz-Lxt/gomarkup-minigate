# 原始需求 Prompt（存档）

> 存档时间：2026-08-20 13:53 (GMT+8)

使用go语言实现一个轻量级 API 网关（类似 Mini Kong / APISIX）需要有一个后台管理页面

具体功能：
- **动态路由**：基于前缀树（Trie Tree）或基数树（Radix Tree）实现高性能的 HTTP 路由匹配。
- **反向代理**：接收请求并转发给后端真实服务，支持轮询、随机、加权等负载均衡算法。
- **中间件流水线**：实现 Plugin/Middleware 机制，支持动态加载认证（JWT）、限流（Token Bucket/Leaky Bucket）、日志记录。
- **配置热更新**：支持监听文件变化（或集成 etcd/Consul），在不重启服务的情况下动态更新路由表。
