package health

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJoinHealthURL(t *testing.T) {
	if got := JoinHealthURL("http://a:1/", "health"); got != "http://a:1/health" {
		t.Fatal(got)
	}
}

func TestProbeOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Fatalf("path %s", r.URL.Path)
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()
	if err := Probe(srv.Client(), srv.URL, "/health", 200); err != nil {
		t.Fatal(err)
	}
}

func TestProbeMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	}))
	defer srv.Close()
	if err := Probe(srv.Client(), srv.URL, "/health", 200); err == nil {
		t.Fatal("expected error")
	}
}

func TestAcceptable(t *testing.T) {
	if !Acceptable(200, 0) || Acceptable(503, 0) || !Acceptable(204, 204) {
		t.Fatal("acceptable logic")
	}
}
