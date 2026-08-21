package router

import (
	"fmt"
	"testing"

	"minigate/internal/model"
)

func rt(path string, methods []string, host string, prio int) *model.RouteSpec {
	return &model.RouteSpec{ID: path, Path: path, Methods: methods, Host: host, Enabled: true, Priority: prio, UpstreamID: "u"}
}

func TestRadixStaticAndParam(t *testing.T) {
	root := newNode("", ntStatic)
	root.insert("/echo/ping", []string{"GET"}, rt("/echo/ping", []string{"GET"}, "", 1))
	root.insert("/users/{id}", []string{"GET"}, rt("/users/{id}", []string{"GET"}, "", 1))
	root.insert("/api/*", []string{"GET"}, rt("/api/*", []string{"GET"}, "", 1))

	r, _ := root.match("GET", "/echo/ping", "")
	if r == nil || r.Path != "/echo/ping" {
		t.Fatalf("static miss: %+v", r)
	}
	r, params := root.match("GET", "/users/42", "")
	if r == nil || params["id"] != "42" {
		t.Fatalf("param miss: %+v %v", r, params)
	}
	r, params = root.match("GET", "/api/foo/bar", "")
	if r == nil || params["*"] != "foo/bar" {
		t.Fatalf("wildcard miss: %+v %v", r, params)
	}
	r, _ = root.match("POST", "/echo/ping", "")
	if r != nil {
		t.Fatalf("method should miss")
	}
}

func TestRadixHostAndPriority(t *testing.T) {
	root := newNode("", ntStatic)
	root.insert("/v1/items", []string{"GET"}, rt("/v1/items", []string{"GET"}, "api.local", 5))
	root.insert("/v1/items", []string{"GET"}, &model.RouteSpec{ID: "generic", Path: "/v1/items", Methods: []string{"GET"}, Enabled: true, Priority: 1, UpstreamID: "u"})
	r, _ := root.match("GET", "/v1/items", "api.local")
	if r == nil || r.Host != "api.local" {
		t.Fatalf("host match failed: %+v", r)
	}
}

func TestTableSwapNoWriteLockOnMatch(t *testing.T) {
	tab := NewTable()
	cfg := &model.GatewayConfig{
		Listen: ":8080", AdminListen: ":8081",
		Upstreams: []model.UpstreamSpec{{ID: "u", Name: "u", Algorithm: "round_robin", Nodes: []model.NodeSpec{{Target: "http://127.0.0.1:9", Weight: 1}}}},
		Routes:    []model.RouteSpec{{ID: "r1", Name: "r1", Path: "/ok", Methods: []string{"GET"}, UpstreamID: "u", Enabled: true}},
	}
	tab.Swap(cfg)
	r, _ := tab.Match("GET", "/ok", "")
	if r == nil {
		t.Fatal("expected match")
	}
	cfg2 := *cfg
	cfg2.Routes = []model.RouteSpec{{ID: "r2", Name: "r2", Path: "/ok", Methods: []string{"GET"}, UpstreamID: "u", Enabled: true}}
	tab.Swap(&cfg2)
	r, _ = tab.Match("GET", "/ok", "")
	if r == nil || r.ID != "r2" {
		t.Fatalf("swap not visible: %+v", r)
	}
}

func BenchmarkMatch1000(b *testing.B) {
	root := newNode("", ntStatic)
	for i := 0; i < 1000; i++ {
		p := fmt.Sprintf("/svc/%d/items/{id}", i)
		root.insert(p, []string{"GET"}, rt(p, []string{"GET"}, "", 1))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		root.match("GET", "/svc/42/items/xyz", "")
	}
}
