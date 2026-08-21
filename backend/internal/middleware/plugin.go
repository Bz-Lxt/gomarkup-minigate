package middleware

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"minigate/internal/model"
)

type Plugin interface {
	Name() string
	Handle(next http.Handler, cfg map[string]any) http.Handler
	ValidateConfig(cfg map[string]any) error
}

var registry = map[string]Plugin{}

func Register(p Plugin) {
	registry[p.Name()] = p
}

func Get(name string) (Plugin, bool) {
	p, ok := registry[name]
	return p, ok
}

func Names() []string {
	out := make([]string, 0, len(registry))
	for k := range registry {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func ValidateNamed(name string, cfg map[string]any) error {
	p, ok := registry[name]
	if !ok {
		return fmt.Errorf("unknown middleware %q", name)
	}
	if cfg == nil {
		cfg = map[string]any{}
	}
	return p.ValidateConfig(cfg)
}

type chainItem struct {
	plugin Plugin
	cfg    map[string]any
}

func BuildChain(next http.Handler, globals []model.MiddlewareSpec, routeMWs []string, lookup func(string) (model.MiddlewareSpec, bool)) http.Handler {
	items := make([]chainItem, 0, len(globals)+len(routeMWs))
	seen := map[string]bool{}
	for _, g := range globals {
		if !g.Enabled {
			continue
		}
		p, ok := registry[g.Name]
		if !ok {
			continue
		}
		items = append(items, chainItem{plugin: p, cfg: g.Config})
		seen[g.Name] = true
	}
	for _, name := range routeMWs {
		if seen[name] {
			continue
		}
		cfg := map[string]any{}
		if spec, ok := lookup(name); ok && spec.Config != nil {
			cfg = spec.Config
		}
		p, ok := registry[name]
		if !ok {
			continue
		}
		items = append(items, chainItem{plugin: p, cfg: cfg})
	}
	h := next
	for i := len(items) - 1; i >= 0; i-- {
		h = items[i].plugin.Handle(h, items[i].cfg)
	}
	return h
}

func strVal(cfg map[string]any, key, def string) string {
	if cfg == nil {
		return def
	}
	v, ok := cfg[key]
	if !ok || v == nil {
		return def
	}
	s := strings.TrimSpace(fmt.Sprint(v))
	if s == "" {
		return def
	}
	return s
}

func floatVal(cfg map[string]any, key string, def float64) float64 {
	if cfg == nil {
		return def
	}
	v, ok := cfg[key]
	if !ok || v == nil {
		return def
	}
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	default:
		var f float64
		_, err := fmt.Sscanf(fmt.Sprint(t), "%f", &f)
		if err != nil {
			return def
		}
		return f
	}
}

func strSlice(cfg map[string]any, key string) []string {
	if cfg == nil {
		return nil
	}
	v, ok := cfg[key]
	if !ok || v == nil {
		return nil
	}
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, x := range t {
			out = append(out, strings.TrimSpace(fmt.Sprint(x)))
		}
		return out
	default:
		return nil
	}
}
