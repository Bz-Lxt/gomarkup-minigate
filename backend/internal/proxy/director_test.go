package proxy

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"minigate/internal/model"
)

func TestApplyDirectorPreservesQuery(t *testing.T) {
	target, _ := url.Parse("http://upstream.local:9001")
	route := &model.RouteSpec{ID: "r1", StripPrefix: "/api"}
	params := map[string]string{"id": "42"}

	req := httptest.NewRequest("GET", "/api/users?page=1&size=20&status=active", nil)
	applyDirector(req, target, route, params)

	if got := req.URL.Path; got != "/users" {
		t.Fatalf("path: got %q want /users", got)
	}
	if got := req.URL.RawQuery; got != "page=1&size=20&status=active" {
		t.Fatalf("raw query lost: got %q", got)
	}
	if got := req.URL.Scheme; got != "http" {
		t.Fatalf("scheme: got %q", got)
	}
	if got := req.Host; got != "upstream.local:9001" {
		t.Fatalf("host: got %q", got)
	}
	if got := req.Header.Get("X-Minigate-Route"); got != "r1" {
		t.Fatalf("route header: got %q", got)
	}
	if got := req.Header.Get("X-Minigate-Param-Id"); got != "42" {
		t.Fatalf("param id header: got %q", got)
	}
}

func TestApplyDirectorNoQuery(t *testing.T) {
	target, _ := url.Parse("http://upstream.local:9001")
	route := &model.RouteSpec{ID: "r2"}

	req := httptest.NewRequest("GET", "/items", nil)
	applyDirector(req, target, route, nil)

	if got := req.URL.RawQuery; got != "" {
		t.Fatalf("expected empty query, got %q", got)
	}
}
