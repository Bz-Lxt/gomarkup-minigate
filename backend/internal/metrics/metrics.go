package metrics

import (
	"sync"
	"sync/atomic"
	"time"

	"minigate/internal/model"
	"minigate/internal/timeutil"
)

type Collector struct {
	total  atomic.Uint64
	mu     sync.Mutex
	bucket [60]uint64
	slot   int
	last   time.Time
	errs   []model.ErrorEvent
}

func New() *Collector {
	c := &Collector{last: time.Now()}
	go c.tick()
	return c
}

func (c *Collector) tick() {
	t := time.NewTicker(time.Second)
	for range t.C {
		c.mu.Lock()
		c.slot = (c.slot + 1) % 60
		c.bucket[c.slot] = 0
		c.last = time.Now()
		c.mu.Unlock()
	}
}

func (c *Collector) Hit() {
	c.total.Add(1)
	c.mu.Lock()
	c.bucket[c.slot]++
	c.mu.Unlock()
}

func (c *Collector) Error(msg string) {
	c.mu.Lock()
	c.errs = append(c.errs, model.ErrorEvent{Time: timeutil.Format(timeutil.Now()), Message: msg})
	if len(c.errs) > 20 {
		c.errs = c.errs[len(c.errs)-20:]
	}
	c.mu.Unlock()
}

func (c *Collector) QPS() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	var sum uint64
	for _, n := range c.bucket {
		sum += n
	}
	return float64(sum) / 60.0
}

func (c *Collector) Total() uint64 { return c.total.Load() }

func (c *Collector) Errors() []model.ErrorEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]model.ErrorEvent, len(c.errs))
	copy(out, c.errs)
	return out
}
