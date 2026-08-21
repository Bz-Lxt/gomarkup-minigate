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

// Routes returns a deep copy of the current route list. Callers can sort,
// reorder, or normalize fields without mutating the live snapshot.
func (t *Table) Routes() []model.RouteSpec {
	s := t.cur.Load()
	if s == nil {
		return nil
	}
	return cloneRouteSpecs(s.routes)
}

// Upstreams returns a deep copy of the current upstream list. Callers can
// safely sort or modify the returned slice and its element fields (including
// the Nodes sub-slice) without mutating the live snapshot.
func (t *Table) Upstreams() []model.UpstreamSpec {
	s := t.cur.Load()
	if s == nil {
		return nil
	}
	out := make([]model.UpstreamSpec, 0, len(s.upstreams))
	for _, u := range s.upstreams {
		out = append(out, cloneUpstreamSpec(u))
	}
	return out
}

func (t *Table) Upstream(id string) (model.UpstreamSpec, bool) {
	s := t.cur.Load()
	if s == nil {
		return model.UpstreamSpec{}, false
	}
	u, ok := s.upstreams[id]
	if !ok {
		return model.UpstreamSpec{}, false
	}
	return cloneUpstreamSpec(u), true
}

// Globals returns a deep copy of the current global middleware list.
func (t *Table) Globals() []model.MiddlewareSpec {
	s := t.cur.Load()
	if s == nil {
		return nil
	}
	return cloneMiddlewareSpecs(s.globals)
}

func (t *Table) Config() model.GatewayConfig {
	s := t.cur.Load()
	if s == nil {
		return model.GatewayConfig{}
	}
	return model.GatewayConfig{
		Listen:            s.listen,
		AdminListen:       s.admin,
		LogLevel:          s.logLevel,
		GlobalMiddlewares: t.Globals(),
		Upstreams:         t.Upstreams(),
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

// cloneRouteSpecs returns a deep copy of the given route slice so callers can
// sort, reorder, or normalize fields (including the Methods sub-slice) without
// mutating the live snapshot's data.
func cloneRouteSpecs(src []model.RouteSpec) []model.RouteSpec {
	if src == nil {
		return nil
	}
	out := make([]model.RouteSpec, len(src))
	for i, r := range src {
		out[i] = r
		if r.Methods != nil {
			out[i].Methods = append([]string(nil), r.Methods...)
		}
		if r.Middlewares != nil {
			out[i].Middlewares = append([]string(nil), r.Middlewares...)
		}
	}
	return out
}

// cloneUpstreamSpec returns a deep copy of the given upstream spec so callers
// can safely mutate the Nodes sub-slice and Config map without affecting the
// live snapshot.
func cloneUpstreamSpec(u model.UpstreamSpec) model.UpstreamSpec {
	out := u
	if u.Nodes != nil {
		out.Nodes = append([]model.NodeSpec(nil), u.Nodes...)
	}
	return out
}

// cloneMiddlewareSpecs returns a deep copy of the given middleware slice so
// callers can safely mutate the Config map of each element without affecting
// the live snapshot.
func cloneMiddlewareSpecs(src []model.MiddlewareSpec) []model.MiddlewareSpec {
	if src == nil {
		return nil
	}
	out := make([]model.MiddlewareSpec, len(src))
	for i, m := range src {
		out[i] = m
		if m.Config != nil {
			out[i].Config = cloneStringAnyMap(m.Config)
		}
	}
	return out
}

// cloneStringAnyMap returns a deep copy of a map[string]any. It copies nested
// maps and slices recursively so mutation of the copy cannot reach the
// original. Non-map/slice values are copied by value as normal.
func cloneStringAnyMap(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = cloneAny(v)
	}
	return dst
}

func cloneAny(v any) any {
	switch val := v.(type) {
	case map[string]any:
		return cloneStringAnyMap(val)
	case []any:
		out := make([]any, len(val))
		for i, e := range val {
			out[i] = cloneAny(e)
		}
		return out
	default:
		return v
	}
}
