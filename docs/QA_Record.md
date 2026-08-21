# QA Record

## Round 1 · 2026-08-20 14:18 GMT+8

**Cost**: ¥0（无计量 API，全程 Mock/离线）

### 环境

`docker compose up --build -d` 成功。全部容器 healthy。

| 检查 | 结果 |
|---|---|
| Docker Build | PASS |
| Health Check（容器内 wget admin /echo） | PASS |
| Go unit tests `go test ./...` | PASS（balancer / config / middleware / router） |
| Radix Benchmark 1000 routes | PASS，3920 ns/op ≪ 1ms |
| API Smoke `tests/api_smoke.py` | PASS 5/5 |
| Admin UI localhost:18482 | PASS（仪表盘 + 路由页可渲染） |

### Smoke 明细

```
tests/api_smoke.py::test_health PASSED
tests/api_smoke.py::test_stats_and_config PASSED
tests/api_smoke.py::test_echo_proxy_round_robin PASSED
tests/api_smoke.py::test_jwt_protect_and_demo_token PASSED
tests/api_smoke.py::test_route_crud_hot_reload PASSED
```

容器内复核：

```
docker compose exec gateway wget -qO- http://127.0.0.1:8081/api/v1/health  -> up
docker compose exec gateway wget -qO- http://127.0.0.1:8080/echo/ping     -> instance=alpha
```

### 错误日志

无。

### 结论

Round 1 PASS，进入 Phase 5 审计。
