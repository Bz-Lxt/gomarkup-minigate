package proxy_test

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"minigate/internal/balancer"
	"minigate/internal/metrics"
	"minigate/internal/model"
	"minigate/internal/proxy"
	"minigate/internal/router"
)

func TestEngineRetriesTransportError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	failedTarget := "http://" + ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}

	var healthyRequests atomic.Int32
	healthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		healthyRequests.Add(1)
		_, _ = io.WriteString(w, "from-healthy-node")
	}))
	defer healthy.Close()

	cfg := &model.GatewayConfig{
		Upstreams: []model.UpstreamSpec{{
			ID:            "api",
			Algorithm:     "round_robin",
			TimeoutMS:     500,
			FailThreshold: 3,
			Nodes: []model.NodeSpec{
				{Target: failedTarget},
				{Target: healthy.URL},
			},
		}},
		Routes: []model.RouteSpec{{
			ID: "read-api", Path: "/items", Methods: []string{http.MethodGet},
			UpstreamID: "api", Enabled: true,
		}},
	}
	table := router.NewTable()
	table.Swap(cfg)
	lb := balancer.NewRegistry()
	lb.Rebuild(cfg.Upstreams)
	engine := &proxy.Engine{Table: table, LB: lb, Metric: metrics.New()}

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://gateway/items", nil))

	res := rec.Result()
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK || string(body) != "from-healthy-node" {
		t.Fatalf("GET through gateway = status %d, body %q; want retried healthy response", res.StatusCode, body)
	}
	if got := healthyRequests.Load(); got != 1 {
		t.Fatalf("healthy upstream requests = %d; want 1", got)
	}
}
