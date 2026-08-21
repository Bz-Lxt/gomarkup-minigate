package router

import (
	"strings"
	"sync/atomic"

	"minigate/internal/model"
)

type snapshot struct {
	root      *node
	routes    []model.RouteSpec
	upstreams map[string]model.UpstreamSpec
	globals   []model.MiddlewareSpec
	listen    string
	admin     string
	logLevel  string
}

type Table struct {
	cur atomic.Pointer[snapshot]
}

func NewTable() *Table {
	t := &Table{}
	t.cur.Store(&snapshot{
		root:      newNode("", ntStatic),
		upstreams: map[string]model.UpstreamSpec{},
	})
	return t
}

func BuildSnapshot(cfg *model.GatewayConfig) *snapshot {
	root := newNode("", ntStatic)
	for i := range cfg.Routes {
		r := cfg.Routes[i]
		if !r.Enabled {
			continue
		}
		root.insert(normalizePath(r.Path), r.Methods, &cfg.Routes[i])
	}
	ups := make(map[string]model.UpstreamSpec, len(cfg.Upstreams))
	for _, u := range cfg.Upstreams {
		ups[u.ID] = u
	}
	routes := make([]model.RouteSpec, len(cfg.Routes))
	copy(routes, cfg.Routes)
	globals := make([]model.MiddlewareSpec, len(cfg.GlobalMiddlewares))
	copy(globals, cfg.GlobalMiddlewares)
	return &snapshot{
		root:      root,
		routes:    routes,
		upstreams: ups,
		globals:   globals,
		listen:    cfg.Listen,
		admin:     cfg.AdminListen,
		logLevel:  cfg.LogLevel,
	}
}

func (t *Table) Swap(cfg *model.GatewayConfig) {
	t.cur.Store(BuildSnapshot(cfg))
}

func (t *Table) Match(method, path, host string) (*model.RouteSpec, map[string]string) {
	s := t.cur.Load()
	if s == nil || s.root == nil {
		return nil, nil
	}
	return s.root.match(strings.ToUpper(method), path, host)
}

func (t *Table) Routes() []model.RouteSpec {
	s := t.cur.Load()
	if s == nil {
		return nil
	}
	return s.routes
}

func (t *Table) Upstreams() []model.UpstreamSpec {
	s := t.cur.Load()
	if s == nil {
		return nil
	}
	out := make([]model.UpstreamSpec, 0, len(s.upstreams))
	for _, u := range s.upstreams {
		out = append(out, u)
	}
	return out
}

func (t *Table) Upstream(id string) (model.UpstreamSpec, bool) {
	s := t.cur.Load()
	if s == nil {
		return model.UpstreamSpec{}, false
	}
	u, ok := s.upstreams[id]
	return u, ok
}

func (t *Table) Globals() []model.MiddlewareSpec {
	s := t.cur.Load()
	if s == nil {
		return nil
	}
	out := make([]model.MiddlewareSpec, len(s.globals))
	copy(out, s.globals)
	return out
}

func (t *Table) Config() model.GatewayConfig {
	s := t.cur.Load()
	if s == nil {
		return model.GatewayConfig{}
	}
	ups := make([]model.UpstreamSpec, 0, len(s.upstreams))
	for _, u := range s.upstreams {
		ups = append(ups, u)
	}
	return model.GatewayConfig{
		Listen:            s.listen,
		AdminListen:       s.admin,
		LogLevel:          s.logLevel,
		GlobalMiddlewares: t.Globals(),
		Upstreams:         ups,
		Routes:            t.Routes(),
	}
}

func (t *Table) ActiveRouteCount() int {
	n := 0
	for _, r := range t.Routes() {
		if r.Enabled {
			n++
		}
	}
	return n
}

func normalizePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return p
}
