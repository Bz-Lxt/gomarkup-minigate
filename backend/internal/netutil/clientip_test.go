package netutil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientIPOrder(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.8:1234"
	r.Header.Set("X-Forwarded-For", "1.1.1.1, 2.2.2.2")
	if got := ClientIP(r); got != "1.1.1.1" {
		t.Fatal(got)
	}
}

func TestMatchIPCIDR(t *testing.T) {
	if !MatchIP("10.1.2.3", []string{"10.0.0.0/8"}) {
		t.Fatal("cidr miss")
	}
	if MatchIP("11.0.0.1", []string{"10.0.0.0/8"}) {
		t.Fatal("cidr false positive")
	}
	if !MatchIP("8.8.8.8", []string{"8.8.8.8"}) {
		t.Fatal("exact miss")
	}
}

func TestHostWithoutPort(t *testing.T) {
	if HostWithoutPort("api.local:443") != "api.local" {
		t.Fatal(HostWithoutPort("api.local:443"))
	}
}

func TestClientIPFallback(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "192.168.1.1:80"
	if ClientIP(r) != "192.168.1.1" {
		t.Fatal(ClientIP(r))
	}
	_ = http.StatusOK
}
