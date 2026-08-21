package proxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"minigate/internal/balancer"
	"minigate/internal/circuit"
	"minigate/internal/metrics"
	"minigate/internal/model"
	"minigate/internal/router"
)

// TestCancelPropagatesToUpstream verifies that when the gateway caller cancels
// its request, the upstream's in-flight handler is also cancelled promptly
// (instead of running until the gateway timeout fires).
func TestCancelPropagatesToUpstream(t *testing.T) {
	// upstream: blocks until its request context is cancelled, then records how long it took
	upstreamCancelled := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
		close(upstreamCancelled)
		_, _ = w.Write([]byte("should-not-reach"))
	}))
	defer srv.Close()

	table := router.NewTable()
	table.Swap(&model.GatewayConfig{
		Listen: ":0", AdminListen: ":0",
		Upstreams: []model.UpstreamSpec{{
			ID: "u", Name: "u", Algorithm: "round_robin",
			TimeoutMS: 10000, FailThreshold: 3,
			Nodes: []model.NodeSpec{{Target: srv.URL, Weight: 1}},
		}},
		Routes: []model.RouteSpec{{
			ID: "r", Path: "/slow", Methods: []string{"POST"}, UpstreamID: "u", Enabled: true,
		}},
	})

	lb := balancer.NewRegistry()
	lb.Rebuild(table.Upstreams())
	eng := proxyEngineForTest(table, lb)

	// client request that we will cancel
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequestWithContext(ctx, "POST", "/slow", nil)
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		eng.ServeHTTP(rec, req)
		close(done)
	}()

	// give the gateway a moment to dial upstream and start the handler
	time.Sleep(150 * time.Millisecond)

	// caller cancels — this must promptly cancel the upstream
	cancel()

	select {
	case <-upstreamCancelled:
		// good: upstream saw cancellation quickly
	case <-time.After(1 * time.Second):
		t.Fatal("upstream was not cancelled within 1s of client cancel; " +
			"gateway did not propagate the cancellation signal")
	}

	// gateway handler should also return without writing a 502
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ServeHTTP did not return after client cancel")
	}

	// node should NOT have been penalized for the client cancel
	b, _, _ := lb.Get("u")
	for _, n := range b.Nodes() {
		if n.Fails() > 0 {
			t.Fatalf("node was penalized for client cancel: fails=%d", n.Fails())
		}
	}
}

// TestCancelNoPenalty verifies that cancelling a request does not trip
// the circuit breaker or mark the node as failed.
func TestCancelNoPenalty(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		<-r.Context().Done()
	}))
	defer srv.Close()

	table := router.NewTable()
	cfg := &model.GatewayConfig{
		Listen: ":0", AdminListen: ":0",
		Upstreams: []model.UpstreamSpec{{
			ID: "u", Name: "u", Algorithm: "round_robin",
			TimeoutMS: 10000, FailThreshold: 1,
			Circuit: model.CircuitSpec{
				Enabled: true, FailureThreshold: 1, SuccessThreshold: 1, OpenTimeoutMS: 60000,
			},
			Nodes: []model.NodeSpec{{Target: srv.URL, Weight: 1}},
		}},
		Routes: []model.RouteSpec{{
			ID: "r", Path: "/slow", Methods: []string{"POST"}, UpstreamID: "u", Enabled: true,
		}},
	}
	table.Swap(cfg)

	lb := balancer.NewRegistry()
	lb.Rebuild(table.Upstreams())
	breaks := circuit.NewRegistry()
	breaks.Reset(table.Upstreams())

	eng := proxyEngineForTest(table, lb)
	eng.Circuit = breaks

	// fire 3 requests, all cancelled by the client
	for i := 0; i < 3; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		req := httptest.NewRequestWithContext(ctx, "POST", "/slow", nil)
		rec := httptest.NewRecorder()
		done := make(chan struct{})
		go func() {
			eng.ServeHTTP(rec, req)
			close(done)
		}()
		time.Sleep(50 * time.Millisecond)
		cancel()
		<-done
	}

	time.Sleep(100 * time.Millisecond)

	// circuit should still be closed (not tripped by client cancels)
	st := breaks.Status()
	if st["u"] != "closed" {
		t.Fatalf("circuit should be closed after client cancels, got %s", st["u"])
	}

	// node should not be marked down
	b, _, _ := lb.Get("u")
	for _, n := range b.Nodes() {
		if !n.Healthy() {
			t.Fatalf("node should still be healthy after client cancels")
		}
	}
}

// TestNormalRequestStillWorks ensures the cancel-propagation change doesn't
// break the happy path.
func TestNormalRequestStillWorks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test", "yes")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	table := router.NewTable()
	table.Swap(&model.GatewayConfig{
		Listen: ":0", AdminListen: ":0",
		Upstreams: []model.UpstreamSpec{{
			ID: "u", Name: "u", Algorithm: "round_robin",
			TimeoutMS: 5000, FailThreshold: 3,
			Nodes: []model.NodeSpec{{Target: srv.URL, Weight: 1}},
		}},
		Routes: []model.RouteSpec{{
			ID: "r", Path: "/ok", Methods: []string{"GET"}, UpstreamID: "u", Enabled: true,
		}},
	})

	lb := balancer.NewRegistry()
	lb.Rebuild(table.Upstreams())
	eng := proxyEngineForTest(table, lb)

	req := httptest.NewRequest("GET", "/ok", nil)
	rec := httptest.NewRecorder()
	eng.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != `{"ok":true}` {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func proxyEngineForTest(table *router.Table, lb *balancer.Registry) *Engine {
	return &Engine{Table: table, LB: lb, Metric: metrics.New()}
}
