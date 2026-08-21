package middleware

import (
	"fmt"
	"net/http"
)

func init() {
	Register(&HeaderPlugin{})
}

type HeaderPlugin struct{}

func (p *HeaderPlugin) Name() string { return "headers" }

func (p *HeaderPlugin) ValidateConfig(cfg map[string]any) error {
	if cfg == nil {
		return nil
	}
	for _, key := range []string{"request_set", "response_set"} {
		if v, ok := cfg[key]; ok && v != nil {
			if _, ok := v.(map[string]any); !ok {
				return fmt.Errorf("%s must be an object", key)
			}
		}
	}
	return nil
}

func mapString(cfg map[string]any, key string) map[string]string {
	out := map[string]string{}
	if cfg == nil {
		return out
	}
	raw, ok := cfg[key]
	if !ok || raw == nil {
		return out
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return out
	}
	for k, v := range m {
		out[k] = fmt.Sprint(v)
	}
	return out
}

func (p *HeaderPlugin) Handle(next http.Handler, cfg map[string]any) http.Handler {
	reqSet := mapString(cfg, "request_set")
	respSet := mapString(cfg, "response_set")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for k, v := range reqSet {
			r.Header.Set(k, v)
		}
		hw := &headerWriter{ResponseWriter: w, extra: respSet}
		next.ServeHTTP(hw, r)
	})
}

type headerWriter struct {
	http.ResponseWriter
	extra map[string]string
	wrote bool
}

func (w *headerWriter) WriteHeader(code int) {
	if !w.wrote {
		for k, v := range w.extra {
			w.Header().Set(k, v)
		}
		w.wrote = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *headerWriter) Write(p []byte) (int, error) {
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(p)
}
