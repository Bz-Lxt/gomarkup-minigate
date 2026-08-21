package middleware

import (
	"fmt"
	"net/http"

	"minigate/internal/netutil"
)

func init() {
	Register(&IPFilterPlugin{})
}

type IPFilterPlugin struct{}

func (p *IPFilterPlugin) Name() string { return "ipfilter" }

func (p *IPFilterPlugin) ValidateConfig(cfg map[string]any) error {
	mode := strVal(cfg, "mode", "allow")
	if mode != "allow" && mode != "deny" {
		return fmt.Errorf("mode must be allow or deny")
	}
	return nil
}

func (p *IPFilterPlugin) Handle(next http.Handler, cfg map[string]any) http.Handler {
	mode := strVal(cfg, "mode", "allow")
	list := strSlice(cfg, "list")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := netutil.ClientIP(r)
		hit := netutil.MatchIP(ip, list)
		if mode == "allow" && !hit && len(list) > 0 {
			http.Error(w, `{"code":40301,"message":"ip not allowed"}`, http.StatusForbidden)
			return
		}
		if mode == "deny" && hit {
			http.Error(w, `{"code":40302,"message":"ip denied"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
