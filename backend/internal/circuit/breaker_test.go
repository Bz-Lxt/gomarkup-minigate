package circuit

import (
	"testing"
	"time"

	"minigate/internal/model"
)

func TestTripAndHalfOpen(t *testing.T) {
	b := New(Config{FailureThreshold: 2, SuccessThreshold: 1, OpenTimeout: 20 * time.Millisecond})
	if !b.Allow() {
		t.Fatal("closed should allow")
	}
	b.Failure()
	if st, _ := b.Snapshot(); st != Closed {
		t.Fatal("need 2 failures")
	}
	b.Failure()
	if st, _ := b.Snapshot(); st != Open {
		t.Fatal("should open")
	}
	if b.Allow() {
		t.Fatal("open should reject")
	}
	time.Sleep(25 * time.Millisecond)
	if !b.Allow() {
		t.Fatal("should half-open")
	}
	b.Success()
	if st, _ := b.Snapshot(); st != Closed {
		t.Fatalf("want closed got %s", st)
	}
}

func TestRegistrySkipDisabled(t *testing.T) {
	r := NewRegistry()
	if r.For("u", model.CircuitSpec{Enabled: false}) != nil {
		t.Fatal("disabled must be nil")
	}
	if r.For("u", model.CircuitSpec{Enabled: true}) == nil {
		t.Fatal("enabled must exist")
	}
}

func TestNormalizeDefaults(t *testing.T) {
	c := Normalize(model.CircuitSpec{})
	if c.FailureThreshold != 5 || c.SuccessThreshold != 2 || c.OpenTimeout <= 0 {
		t.Fatalf("%+v", c)
	}
}
