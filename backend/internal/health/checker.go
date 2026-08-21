package health

import (
	"net/http"
	"time"

	"log/slog"
	"minigate/internal/balancer"
	"minigate/internal/logger"
)

type Checker struct {
	LB     *balancer.Registry
	Client *http.Client
}

func New(lb *balancer.Registry) *Checker {
	return &Checker{
		LB:     lb,
		Client: &http.Client{Timeout: 2 * time.Second},
	}
}

func (c *Checker) Start(stop <-chan struct{}) {
	t := time.NewTicker(5 * time.Second)
	go func() {
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				c.probeAll()
			}
		}
	}()
}

func (c *Checker) probeAll() {
	for id, b := range c.LB.All() {
		_, spec, ok := c.LB.Get(id)
		if !ok {
			continue
		}
		path := spec.HealthPath
		if path == "" {
			path = "/health"
		}
		expect := spec.ExpectedStatus
		if expect <= 0 {
			expect = http.StatusOK
		}
		for _, n := range b.Nodes() {
			err := Probe(c.Client, n.Target, path, expect)
			if err != nil {
				n.Report(false, spec.FailThreshold)
				logger.L().Debug("active probe fail", slog.String("target", n.Target), slog.String("err", err.Error()))
				continue
			}
			n.ForceDown(false)
		}
	}
}
