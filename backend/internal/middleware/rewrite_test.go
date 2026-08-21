package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRewriteStripAndAdd(t *testing.T) {
	p := &RewritePlugin{}
	var got string
	h := p.Handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Path
		w.WriteHeader(204)
	}), map[string]any{"strip_prefix": "/api", "add_prefix": "/v2"})
	req := httptest.NewRequest("GET", "/api/users", nil)
	h.ServeHTTP(httptest.NewRecorder(), req)
	if got != "/v2/users" {
		t.Fatalf("got %s", got)
	}
}

func TestRewriteSetPath(t *testing.T) {
	p := &RewritePlugin{}
	var got string
	h := p.Handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Path
	}), map[string]any{"set_path": "/fixed"})
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/anything", nil))
	if got != "/fixed" {
		t.Fatalf("got %s", got)
	}
}

func TestIPFilterDeny(t *testing.T) {
	p := &IPFilterPlugin{}
	h := p.Handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	}), map[string]any{"mode": "deny", "list": []any{"10.0.0.1"}})
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.1:9"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 403 {
		t.Fatalf("code %d", rec.Code)
	}
}
