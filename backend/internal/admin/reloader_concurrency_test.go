package admin_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"minigate/internal/admin"
	"minigate/internal/balancer"
	"minigate/internal/config"
	"minigate/internal/metrics"
	"minigate/internal/model"
	"minigate/internal/router"
)

type memorySource struct{}

func (memorySource) Name() string { return "memory" }

func (memorySource) Load() (*model.GatewayConfig, error) {
	return &model.GatewayConfig{}, nil
}

func (memorySource) Save(*model.GatewayConfig) error { return nil }

func TestStatsRemainResponsiveDuringConfigApply(t *testing.T) {
	applyStarted := make(chan struct{})
	releaseApply := make(chan struct{})
	applyDone := make(chan error, 1)
	reloader := config.NewReloader(memorySource{}, func(*model.GatewayConfig) error {
		close(applyStarted)
		<-releaseApply
		return nil
	})

	handler := (&admin.Server{
		Table:    router.NewTable(),
		LB:       balancer.NewRegistry(),
		Metric:   metrics.New(),
		Reloader: reloader,
	}).Handler()

	go func() {
		applyDone <- reloader.SaveAndApply(&model.GatewayConfig{})
	}()

	select {
	case <-applyStarted:
	case <-time.After(time.Second):
		t.Fatal("configuration apply did not start")
	}

	responseDone := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil))
		responseDone <- recorder
	}()

	select {
	case recorder := <-responseDone:
		if recorder.Code != http.StatusOK {
			t.Errorf("GET /api/v1/stats status = %d, want %d", recorder.Code, http.StatusOK)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("GET /api/v1/stats blocked while configuration was applying")
	}

	close(releaseApply)
	if err := <-applyDone; err != nil {
		t.Fatalf("apply configuration: %v", err)
	}
}
