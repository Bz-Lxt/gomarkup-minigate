package metrics_test

import (
	"testing"
	"time"

	"minigate/internal/metrics"
)

func TestCollectorRemainsResponsiveAcrossBucketRotation(t *testing.T) {
	c := metrics.New()
	c.Hit()

	time.Sleep(1100 * time.Millisecond)

	done := make(chan struct{})
	go func() {
		c.Hit()
		_ = c.QPS()
		_ = c.Errors()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("collector stopped responding after its time bucket rotated")
	}
}
