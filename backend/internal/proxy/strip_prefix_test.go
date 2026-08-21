package proxy_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"minigate/internal/balancer"
	"minigate/internal/metrics"
	"minigate/internal/model"
	"minigate/internal/proxy"
	"minigate/internal/router"
)

func TestStripPrefixAtRouteRoot(t *testing.T) {
	paths := make(chan string, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths <- r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()

	cfg := &model.GatewayConfig{
		Routes: []model.RouteSpec{{
			ID:          "api",
			Path:        "/api",
			UpstreamID:  "service",
			Enabled:     true,
			StripPrefix: "/api",
		}},
		Upstreams: []model.UpstreamSpec{{
			ID:            "service",
			FailThreshold: 3,
			Nodes:         []model.NodeSpec{{Target: upstream.URL}},
		}},
	}
	table := router.NewTable()
	table.Swap(cfg)
	lb := balancer.NewRegistry()
	lb.Rebuild(cfg.Upstreams)
	engine := &proxy.Engine{Table: table, LB: lb, Metric: metrics.New()}

	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/api", nil)
	res := httptest.NewRecorder()
	engine.ServeHTTP(res, req)
	result := res.Result()
	defer result.Body.Close()
	_, _ = io.Copy(io.Discard, result.Body)

	if result.StatusCode != http.StatusNoContent {
		t.Fatalf("gateway status = %d, want %d", result.StatusCode, http.StatusNoContent)
	}
	if got := <-paths; got != "/" {
		t.Fatalf("upstream path = %q, want %q", got, "/")
	}
}
