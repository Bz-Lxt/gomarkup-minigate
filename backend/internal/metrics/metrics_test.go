package metrics

import (
	"sync"
	"testing"
	"time"
)

func TestTickDoesNotDeadlock(t *testing.T) {
	c := New()
	// Wait for at least two ticks to ensure the second iteration also unlocks.
	time.Sleep(2500 * time.Millisecond)

	// If tick() defers unlock, the second iteration would still hold the lock
	// and these calls would block forever.
	const workers = 50
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			c.Hit()
			_ = c.QPS()
			_ = c.Errors()
		}()
	}
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("metrics Collector deadlocked after bucket rotation")
	}

	if got := c.Total(); got != workers {
		t.Fatalf("total = %d, want %d", got, workers)
	}
}
