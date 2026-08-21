package middleware

import (
	"testing"
	"time"
)

func TestTokenBucketRate(t *testing.T) {
	b := NewTokenBucket(100, 100)
	ok := 0
	for i := 0; i < 100; i++ {
		if b.Allow() {
			ok++
		}
	}
	if ok != 100 {
		t.Fatalf("burst allow %d", ok)
	}
	if b.Allow() {
		t.Fatal("should be empty")
	}
	time.Sleep(20 * time.Millisecond)
	if !b.Allow() {
		t.Fatal("should refill")
	}
}

func TestLeakyBucketCapacity(t *testing.T) {
	b := NewLeakyBucket(1000, 5)
	ok := 0
	for i := 0; i < 10; i++ {
		if b.Allow() {
			ok++
		}
	}
	if ok != 5 {
		t.Fatalf("leaky allow %d want 5", ok)
	}
}

func TestRateLimitValidate(t *testing.T) {
	p := &RateLimitPlugin{}
	if err := p.ValidateConfig(map[string]any{"algorithm": "nope", "rate": 1}); err == nil {
		t.Fatal("expected invalid algo")
	}
	if err := p.ValidateConfig(map[string]any{"algorithm": "token_bucket", "rate": 10}); err != nil {
		t.Fatal(err)
	}
}
