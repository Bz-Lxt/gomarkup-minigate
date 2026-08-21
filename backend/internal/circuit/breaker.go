package circuit

import (
	"sync"
	"time"

	"minigate/internal/model"
)

type State int

const (
	Closed State = iota
	Open
	HalfOpen
)

func (s State) String() string {
	switch s {
	case Open:
		return "open"
	case HalfOpen:
		return "half_open"
	default:
		return "closed"
	}
}

type Config struct {
	FailureThreshold int
	SuccessThreshold int
	OpenTimeout      time.Duration
}

func Normalize(spec model.CircuitSpec) Config {
	c := Config{
		FailureThreshold: spec.FailureThreshold,
		SuccessThreshold: spec.SuccessThreshold,
		OpenTimeout:      time.Duration(spec.OpenTimeoutMS) * time.Millisecond,
	}
	if c.FailureThreshold <= 0 {
		c.FailureThreshold = 5
	}
	if c.SuccessThreshold <= 0 {
		c.SuccessThreshold = 2
	}
	if c.OpenTimeout <= 0 {
		c.OpenTimeout = 8 * time.Second
	}
	return c
}

type Breaker struct {
	mu        sync.Mutex
	cfg       Config
	state     State
	failures  int
	successes int
	openedAt  time.Time
}

func New(cfg Config) *Breaker {
	return &Breaker{cfg: cfg, state: Closed}
}

func (b *Breaker) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch b.state {
	case Open:
		if time.Since(b.openedAt) >= b.cfg.OpenTimeout {
			b.state = HalfOpen
			b.successes = 0
			return true
		}
		return false
	default:
		return true
	}
}

func (b *Breaker) Success() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures = 0
	if b.state == HalfOpen {
		b.successes++
		if b.successes >= b.cfg.SuccessThreshold {
			b.state = Closed
			b.successes = 0
		}
	}
}

func (b *Breaker) Failure() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.state == HalfOpen {
		b.tripLocked()
		return
	}
	b.failures++
	if b.failures >= b.cfg.FailureThreshold {
		b.tripLocked()
	}
}

func (b *Breaker) tripLocked() {
	b.state = Open
	b.openedAt = time.Now()
	b.failures = 0
	b.successes = 0
}

func (b *Breaker) Snapshot() (State, int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.state, b.failures
}
