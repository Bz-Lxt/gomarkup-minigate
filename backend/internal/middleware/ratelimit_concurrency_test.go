package middleware_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"minigate/internal/middleware"
)

func TestRateLimitConcurrentFirstRequestsShareBurst(t *testing.T) {
	previousProcs := runtime.GOMAXPROCS(4)
	t.Cleanup(func() { runtime.GOMAXPROCS(previousProcs) })

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := (&middleware.RateLimitPlugin{}).Handle(next, map[string]any{
		"algorithm": "token_bucket",
		"dimension": "route",
		"rate":      0.000000001,
		"burst":     1,
	})

	const (
		rounds  = 50
		workers = 64
	)
	for round := 0; round < rounds; round++ {
		start := make(chan struct{})
		statuses := make(chan int, workers)
		path := fmt.Sprintf("/reports/%d", round)

		for i := 0; i < workers; i++ {
			go func() {
				req := httptest.NewRequest(http.MethodGet, path, nil)
				rec := httptest.NewRecorder()
				<-start
				handler.ServeHTTP(rec, req)
				statuses <- rec.Code
			}()
		}
		close(start)

		allowed := 0
		for i := 0; i < workers; i++ {
			switch status := <-statuses; status {
			case http.StatusNoContent:
				allowed++
			case http.StatusTooManyRequests:
			default:
				t.Fatalf("path %s returned unexpected status %d", path, status)
			}
		}
		if allowed != 1 {
			t.Fatalf("path %s allowed %d concurrent requests with burst=1; want 1", path, allowed)
		}
	}
}
