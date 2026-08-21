package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"minigate/internal/netutil"
)

func init() {
	Register(&RateLimitPlugin{})
}

type RateLimitPlugin struct{}

func (p *RateLimitPlugin) Name() string { return "ratelimit" }

func (p *RateLimitPlugin) ValidateConfig(cfg map[string]any) error {
	algo := strVal(cfg, "algorithm", "token_bucket")
	if algo != "token_bucket" && algo != "leaky_bucket" {
		return fmt.Errorf("algorithm must be token_bucket or leaky_bucket")
	}
	if floatVal(cfg, "rate", 100) <= 0 {
		return fmt.Errorf("rate must be > 0")
	}
	return nil
}

func (p *RateLimitPlugin) Handle(next http.Handler, cfg map[string]any) http.Handler {
	algo := strVal(cfg, "algorithm", "token_bucket")
	rate := floatVal(cfg, "rate", 100)
	burst := floatVal(cfg, "burst", rate)
	if burst < 1 {
		burst = 1
	}
	dim := strVal(cfg, "dimension", "ip")
	store := newLimiterStore(algo, rate, burst)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := dimKey(dim, r)
		if !store.Allow(key) {
			w.Header().Set("Retry-After", "1")
			http.Error(w, `{"code":42901,"message":"rate limited"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func dimKey(dim string, r *http.Request) string {
	switch dim {
	case "global":
		return "global"
	case "route":
		return "route:" + r.URL.Path
	default:
		return "ip:" + netutil.ClientIP(r)
	}
}

type limiter interface {
	Allow() bool
}

type tokenBucket struct {
	mu     sync.Mutex
	rate   float64
	burst  float64
	tokens float64
	last   time.Time
}

func (b *tokenBucket) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	if !b.last.IsZero() {
		b.tokens += now.Sub(b.last).Seconds() * b.rate
		if b.tokens > b.burst {
			b.tokens = b.burst
		}
	} else {
		b.tokens = b.burst
	}
	b.last = now
	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

type leakyBucket struct {
	mu       sync.Mutex
	rate     float64
	capacity float64
	water    float64
	last     time.Time
}

func (b *leakyBucket) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	if !b.last.IsZero() {
		b.water -= now.Sub(b.last).Seconds() * b.rate
		if b.water < 0 {
			b.water = 0
		}
	}
	b.last = now
	if b.water+1 <= b.capacity {
		b.water++
		return true
	}
	return false
}

type limiterStore struct {
	mu    sync.Mutex
	algo  string
	rate  float64
	burst float64
	items map[string]limiter
}

func newLimiterStore(algo string, rate, burst float64) *limiterStore {
	return &limiterStore{algo: algo, rate: rate, burst: burst, items: map[string]limiter{}}
}

func (s *limiterStore) Allow(key string) bool {
	s.mu.Lock()
	l, ok := s.items[key]
	s.mu.Unlock()
	if !ok {
		if s.algo == "leaky_bucket" {
			l = &leakyBucket{rate: s.rate, capacity: s.burst}
		} else {
			l = &tokenBucket{rate: s.rate, burst: s.burst, tokens: s.burst}
		}
		s.mu.Lock()
		s.items[key] = l
		s.mu.Unlock()
	}
	return l.Allow()
}

func NewTokenBucket(rate, burst float64) *tokenBucket {
	return &tokenBucket{rate: rate, burst: burst, tokens: burst}
}

func NewLeakyBucket(rate, capacity float64) *leakyBucket {
	return &leakyBucket{rate: rate, capacity: capacity}
}
