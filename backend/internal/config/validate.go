package config

import (
	"fmt"
	"strings"

	"minigate/internal/middleware"
	"minigate/internal/model"
)

var validAlgorithms = map[string]struct{}{
	"":            {},
	"round_robin": {},
	"random":      {},
	"weighted_rr": {},
	"least_conn":  {},
}

func Validate(cfg *model.GatewayConfig) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if strings.TrimSpace(cfg.Listen) == "" {
		cfg.Listen = ":8080"
	}
	if strings.TrimSpace(cfg.AdminListen) == "" {
		cfg.AdminListen = ":8081"
	}
	if strings.TrimSpace(cfg.LogLevel) == "" {
		cfg.LogLevel = "info"
	}
	ups := map[string]struct{}{}
	for i, u := range cfg.Upstreams {
		if strings.TrimSpace(u.ID) == "" {
			return fmt.Errorf("upstreams[%d].id is required", i)
		}
		if _, ok := ups[u.ID]; ok {
			return fmt.Errorf("duplicate upstream id %q", u.ID)
		}
		ups[u.ID] = struct{}{}
		if len(u.Nodes) == 0 {
			return fmt.Errorf("upstream %q has no nodes", u.ID)
		}
		if _, ok := validAlgorithms[u.Algorithm]; !ok {
			return fmt.Errorf("upstream %q algorithm %q invalid", u.ID, u.Algorithm)
		}
		for j, n := range u.Nodes {
			if !strings.HasPrefix(n.Target, "http://") && !strings.HasPrefix(n.Target, "https://") {
				return fmt.Errorf("upstream %q nodes[%d].target must be http(s) url", u.ID, j)
			}
		}
		if u.TimeoutMS <= 0 {
			cfg.Upstreams[i].TimeoutMS = 5000
		}
		if u.FailThreshold <= 0 {
			cfg.Upstreams[i].FailThreshold = 3
		}
		if u.HealthPath == "" {
			cfg.Upstreams[i].HealthPath = "/health"
		}
		if u.ExpectedStatus <= 0 {
			cfg.Upstreams[i].ExpectedStatus = 200
		}
		if u.ProbeIntervalMS <= 0 {
			cfg.Upstreams[i].ProbeIntervalMS = 5000
		}
		if u.Circuit.Enabled {
			if u.Circuit.FailureThreshold <= 0 {
				cfg.Upstreams[i].Circuit.FailureThreshold = 5
			}
			if u.Circuit.SuccessThreshold <= 0 {
				cfg.Upstreams[i].Circuit.SuccessThreshold = 2
			}
			if u.Circuit.OpenTimeoutMS <= 0 {
				cfg.Upstreams[i].Circuit.OpenTimeoutMS = 8000
			}
		}
	}
	for _, m := range cfg.GlobalMiddlewares {
		if strings.TrimSpace(m.Name) == "" {
			return fmt.Errorf("global middleware name is required")
		}
		if err := middleware.ValidateNamed(m.Name, m.Config); err != nil {
			return fmt.Errorf("middleware %q: %w", m.Name, err)
		}
	}
	routes := map[string]struct{}{}
	for i, r := range cfg.Routes {
		if strings.TrimSpace(r.ID) == "" {
			return fmt.Errorf("routes[%d].id is required", i)
		}
		if _, ok := routes[r.ID]; ok {
			return fmt.Errorf("duplicate route id %q", r.ID)
		}
		routes[r.ID] = struct{}{}
		if !strings.HasPrefix(r.Path, "/") {
			return fmt.Errorf("route %q path must start with /", r.ID)
		}
		if strings.TrimSpace(r.UpstreamID) == "" {
			return fmt.Errorf("route %q upstream_id is required", r.ID)
		}
		if _, ok := ups[r.UpstreamID]; !ok {
			return fmt.Errorf("route %q references missing upstream %q", r.ID, r.UpstreamID)
		}
		for _, name := range r.Middlewares {
			if _, ok := middleware.Get(name); !ok {
				return fmt.Errorf("route %q unknown middleware %q", r.ID, name)
			}
		}
		if r.Name == "" {
			cfg.Routes[i].Name = r.ID
		}
	}
	return nil
}
