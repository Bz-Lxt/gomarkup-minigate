package middleware

import (
	"net/http"
	"strings"
)

func init() {
	Register(&CORSPlugin{})
}

type CORSPlugin struct{}

func (p *CORSPlugin) Name() string { return "cors" }

func (p *CORSPlugin) ValidateConfig(cfg map[string]any) error {
	_ = cfg
	return nil
}

func (p *CORSPlugin) Handle(next http.Handler, cfg map[string]any) http.Handler {
	origins := strSlice(cfg, "allow_origins")
	if len(origins) == 0 {
		origins = []string{"*"}
	}
	methods := strVal(cfg, "allow_methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
	headers := strVal(cfg, "allow_headers", "Authorization,Content-Type,X-Request-ID")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allow := "*"
		if origins[0] != "*" && origin != "" {
			allow = ""
			for _, o := range origins {
				if strings.EqualFold(o, origin) {
					allow = origin
					break
				}
			}
		} else if origin != "" && origins[0] == "*" {
			allow = "*"
		}
		if allow != "" {
			w.Header().Set("Access-Control-Allow-Origin", allow)
			w.Header().Set("Access-Control-Allow-Methods", methods)
			w.Header().Set("Access-Control-Allow-Headers", headers)
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
