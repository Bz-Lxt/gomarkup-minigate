package proxy_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"minigate/internal/balancer"
	"minigate/internal/metrics"
	"minigate/internal/model"
	"minigate/internal/proxy"
	"minigate/internal/router"
)

func TestProxyPropagatesClientCancellation(t *testing.T) {
	started := make(chan struct{}, 1)
	canceled := make(chan struct{}, 1)
	release := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case started <- struct{}{}:
		default:
		}
		select {
		case <-r.Context().Done():
			select {
			case canceled <- struct{}{}:
			default:
			}
		case <-release:
		}
	}))
	defer upstream.Close()

	cfg := &model.GatewayConfig{
		Upstreams: []model.UpstreamSpec{{
			ID: "slow", TimeoutMS: 2000, FailThreshold: 3,
			Nodes: []model.NodeSpec{{Target: upstream.URL}},
		}},
		Routes: []model.RouteSpec{{
			ID: "slow-route", Path: "/work", Methods: []string{http.MethodPost},
			UpstreamID: "slow", Enabled: true,
		}},
	}
	table := router.NewTable()
	table.Swap(cfg)
	lb := balancer.NewRegistry()
	lb.Rebuild(cfg.Upstreams)
	gateway := httptest.NewServer(&proxy.Engine{Table: table, LB: lb, Metric: metrics.New()})
	defer func() {
		close(release)
		gateway.Close()
	}()

	ctx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, gateway.URL+"/work", nil)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			resp.Body.Close()
		}
		close(done)
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("upstream request did not start")
	}
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("client request did not return after cancellation")
	}
	select {
	case <-canceled:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("upstream request continued after the client canceled")
	}
}
