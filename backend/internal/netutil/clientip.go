package netutil

import (
	"net"
	"net/http"
	"strings"
)

func ClientIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	if xr := strings.TrimSpace(r.Header.Get("X-Real-IP")); xr != "" {
		return xr
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func MatchIP(ip string, rules []string) bool {
	parsed := net.ParseIP(ip)
	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		if rule == "" {
			continue
		}
		if strings.Contains(rule, "/") {
			_, cidr, err := net.ParseCIDR(rule)
			if err != nil || parsed == nil {
				continue
			}
			if cidr.Contains(parsed) {
				return true
			}
			continue
		}
		if strings.EqualFold(ip, rule) {
			return true
		}
	}
	return false
}

func HostWithoutPort(host string) string {
	if i := strings.IndexByte(host, ':'); i >= 0 {
		h, _, err := net.SplitHostPort(host)
		if err == nil {
			return h
		}
		return host[:i]
	}
	return host
}
