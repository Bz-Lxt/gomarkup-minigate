package proxy_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"minigate/internal/balancer"
	"minigate/internal/metrics"
	"minigate/internal/model"
	"minigate/internal/proxy"
	"minigate/internal/router"
)

func TestEnginePreservesClientQueryParameters(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, r.URL.RawQuery)
	}))
	defer upstream.Close()

	cfg := &model.GatewayConfig{
		Upstreams: []model.UpstreamSpec{{
			ID:    "catalog",
			Nodes: []model.NodeSpec{{Target: upstream.URL}},
		}},
		Routes: []model.RouteSpec{{
			ID:         "catalog-list",
			Path:       "/products",
			Methods:    []string{http.MethodGet},
			UpstreamID: "catalog",
			Enabled:    true,
		}},
	}
	table := router.NewTable()
	table.Swap(cfg)
	lb := balancer.NewRegistry()
	lb.Rebuild(cfg.Upstreams)
	engine := &proxy.Engine{Table: table, LB: lb, Metric: metrics.New()}

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://gateway.local/products?category=books&page=3", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got, want := rec.Body.String(), "category=books&page=3"; got != want {
		t.Fatalf("upstream query = %q, want %q", got, want)
	}
}
