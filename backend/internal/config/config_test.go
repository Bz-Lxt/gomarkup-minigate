package config

import (
	"os"
	"path/filepath"
	"testing"

	"minigate/internal/model"

	_ "minigate/internal/middleware"
)

func TestValidateRejectsBadUpstream(t *testing.T) {
	cfg := &model.GatewayConfig{
		Routes: []model.RouteSpec{{ID: "r", Path: "/x", UpstreamID: "missing"}},
	}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected error")
	}
}

func TestFileRoundTripAndHash(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "g.yaml")
	src := &FileSource{Path: p}
	cfg := &model.GatewayConfig{
		Listen: ":1", AdminListen: ":2", LogLevel: "info",
		Upstreams: []model.UpstreamSpec{{
			ID: "u", Name: "u", Algorithm: "round_robin",
			Nodes: []model.NodeSpec{{Target: "http://127.0.0.1:9", Weight: 1}},
		}},
		Routes: []model.RouteSpec{{
			ID: "r", Name: "r", Path: "/x", Methods: []string{"GET"}, UpstreamID: "u", Enabled: true,
		}},
	}
	if err := src.Save(cfg); err != nil {
		t.Fatal(err)
	}
	got, err := src.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Routes[0].ID != "r" {
		t.Fatalf("%+v", got)
	}
	if Hash(cfg) == "" {
		t.Fatal("empty hash")
	}
}

func TestInvalidConfigKeepsOld(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "g.yaml")
	src := &FileSource{Path: p}
	applied := 0
	rl := NewReloader(src, func(cfg *model.GatewayConfig) error {
		applied++
		return nil
	})
	good := &model.GatewayConfig{
		Listen: ":1", AdminListen: ":2",
		Upstreams: []model.UpstreamSpec{{ID: "u", Nodes: []model.NodeSpec{{Target: "http://127.0.0.1:9", Weight: 1}}}},
		Routes:    []model.RouteSpec{{ID: "r", Path: "/x", UpstreamID: "u", Enabled: true}},
	}
	if err := rl.SaveAndApply(good); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("listen: :1\nroutes: [{id: x, path: y}]"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := rl.LoadAndApply(); err == nil {
		t.Fatal("bad yaml/config should fail")
	}
	if applied != 1 {
		t.Fatalf("should keep old snapshot, applied=%d", applied)
	}
	st := rl.Status()
	if st.LastError == "" {
		t.Fatal("expected last_error")
	}
}
