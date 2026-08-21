package proxy

import (
	"testing"

	"minigate/internal/model"
)

func TestRewritePathStripPrefix(t *testing.T) {
	route := &model.RouteSpec{StripPrefix: "/echo"}
	cases := []struct{ in, want string }{
		{"/echo/ping", "/ping"},
		{"/echo", "/"},   // bare root: strip to "/"
		{"/echo/", "/"},  // trailing slash: strip to "/"
		{"/echo/x/y", "/x/y"},
		{"/echoextra", "/echoextra"}, // not a segment boundary, must not strip
	}
	for _, c := range cases {
		got := rewritePath(c.in, route.StripPrefix, "")
		if got != c.want {
			t.Errorf("rewritePath(%q, strip=%q): got %q want %q", c.in, route.StripPrefix, got, c.want)
		}
	}
}

func TestRewritePathStripPrefixWithTargetPath(t *testing.T) {
	cases := []struct{ in, want string }{
		{"/echo", "/"},
		{"/echo/", "/"},
		{"/echo/ping", "/ping"},
	}
	for _, c := range cases {
		got := rewritePath(c.in, "/echo", "")
		if got != c.want {
			t.Errorf("in=%q: got %q want %q", c.in, got, c.want)
		}
	}
}

func TestRewritePathNoStripPrefix(t *testing.T) {
	got := rewritePath("/echo/ping", "", "")
	if got != "/echo/ping" {
		t.Fatalf("got %q want /echo/ping", got)
	}
	// target path prepended when no strip
	got = rewritePath("/echo/ping", "", "/base")
	if got != "/base/echo/ping" {
		t.Fatalf("got %q want /base/echo/ping", got)
	}
}

