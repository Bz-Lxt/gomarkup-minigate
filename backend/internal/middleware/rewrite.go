package middleware

import (
	"fmt"
	"net/http"
	"strings"
)

func init() {
	Register(&RewritePlugin{})
}

type RewritePlugin struct{}

func (p *RewritePlugin) Name() string { return "rewrite" }

func (p *RewritePlugin) ValidateConfig(cfg map[string]any) error {
	set := strVal(cfg, "set_path", "")
	if set != "" && !strings.HasPrefix(set, "/") {
		return fmt.Errorf("set_path must start with /")
	}
	return nil
}

func (p *RewritePlugin) Handle(next http.Handler, cfg map[string]any) http.Handler {
	strip := strVal(cfg, "strip_prefix", "")
	add := strVal(cfg, "add_prefix", "")
	set := strVal(cfg, "set_path", "")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if set != "" {
			path = set
		} else {
			if strip != "" {
				path = strings.TrimPrefix(path, strip)
				if path == "" {
					path = "/"
				}
			}
			if add != "" {
				if !strings.HasPrefix(add, "/") {
					add = "/" + add
				}
				path = strings.TrimRight(add, "/") + path
			}
		}
		r.URL.Path = path
		next.ServeHTTP(w, r)
	})
}
