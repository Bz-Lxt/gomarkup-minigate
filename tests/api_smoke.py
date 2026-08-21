import json
import os
import time
import urllib.error
import urllib.request

ADMIN = os.environ.get("ADMIN_URL", "http://127.0.0.1:18481")
GW = os.environ.get("GATEWAY_URL", "http://127.0.0.1:18480")


def http(method, url, body=None, headers=None, timeout=8):
    data = None if body is None else json.dumps(body).encode()
    req = urllib.request.Request(url, data=data, method=method)
    req.add_header("Content-Type", "application/json")
    for k, v in (headers or {}).items():
        req.add_header(k, v)
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            raw = resp.read().decode()
            return resp.status, json.loads(raw) if raw else {}
    except urllib.error.HTTPError as e:
        raw = e.read().decode()
        try:
            parsed = json.loads(raw) if raw else {}
        except json.JSONDecodeError:
            parsed = {"raw": raw}
        return e.code, parsed


def wait_up():
    for _ in range(40):
        try:
            code, body = http("GET", ADMIN + "/api/v1/health")
            if code == 200 and body.get("data", {}).get("status") == "up":
                code2, _ = http("GET", GW + "/echo/ping")
                if code2 == 200:
                    return
        except Exception:
            time.sleep(1)
    raise AssertionError("admin/gateway not up")


def test_health():
    wait_up()
    code, body = http("GET", ADMIN + "/api/v1/health")
    assert code == 200
    assert body["code"] == 0


def test_stats_and_config():
    code, body = http("GET", ADMIN + "/api/v1/stats")
    assert code == 200
    assert "active_routes" in body["data"]
    code, body = http("GET", ADMIN + "/api/v1/config")
    assert code == 200
    assert body["data"]["listen"]


def test_echo_proxy_round_robin():
    seen = set()
    for _ in range(8):
        code, body = http("GET", GW + "/echo/ping")
        assert code == 200
        seen.add(body["instance"])
    assert seen == {"alpha", "bravo"}


def test_query_params_preserved():
    code, body = http("GET", GW + "/echo/ping?page=2&size=20&status=active")
    assert code == 200
    assert body["query"] == "page=2&size=20&status=active"


def test_jwt_protect_and_demo_token():
    code, _ = http("GET", GW + "/secure/x")
    assert code == 401
    code, body = http("POST", ADMIN + "/api/v1/tokens/demo")
    assert code == 200
    token = body["data"]["token"]
    code, body = http("GET", GW + "/secure/x", headers={"Authorization": "Bearer " + token})
    assert code == 200
    assert body["instance"] in {"alpha", "bravo"}


def test_route_crud_hot_reload():
    rid = "smoke-temp"
    http("DELETE", ADMIN + "/api/v1/routes/" + rid)
    code, _ = http(
        "POST",
        ADMIN + "/api/v1/routes",
        {
            "id": rid,
            "name": "smoke",
            "path": "/smoke-temp",
            "methods": ["GET"],
            "host": "",
            "upstream_id": "echo-rr",
            "middlewares": [],
            "enabled": True,
            "priority": 1,
            "strip_prefix": "",
        },
    )
    assert code in (200, 201)
    time.sleep(0.5)
    code, body = http("GET", GW + "/smoke-temp")
    assert code == 200
    assert body["instance"] in {"alpha", "bravo"}
    code, _ = http("DELETE", ADMIN + "/api/v1/routes/" + rid)
    assert code == 200
    time.sleep(0.5)
    code, _ = http("GET", GW + "/smoke-temp")
    assert code == 404
