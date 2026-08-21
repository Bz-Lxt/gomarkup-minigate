package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"minigate/internal/balancer"
	"minigate/internal/circuit"
	"minigate/internal/config"
	"minigate/internal/metrics"
	"minigate/internal/model"
	"minigate/internal/router"
)

func TestStatsAfterCircuitRegistryReset(t *testing.T) {
	upstreams := []model.UpstreamSpec{{
		ID: "orders",
		Circuit: model.CircuitSpec{
			Enabled:          true,
			FailureThreshold: 2,
			SuccessThreshold: 1,
			OpenTimeoutMS:    1000,
		},
		Nodes: []model.NodeSpec{{Target: "http://127.0.0.1:8080"}},
	}}
	table := router.NewTable()
	table.Swap(&model.GatewayConfig{Upstreams: upstreams})
	lb := balancer.NewRegistry()
	lb.Rebuild(upstreams)
	circuits := circuit.NewRegistry()
	circuits.Reset(upstreams)

	srv := &Server{
		Table:    table,
		LB:       lb,
		Metric:   metrics.New(),
		Reloader: config.NewReloader(&config.FileSource{Path: "unused"}, func(*model.GatewayConfig) error { return nil }),
		Circuit:  circuits,
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	res := httptest.NewRecorder()

	srv.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/stats status = %d, want %d; body = %s", res.Code, http.StatusOK, res.Body.String())
	}
}
