package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTAllowAndDeny(t *testing.T) {
	secret := "test-secret"
	p := &JWTPlugin{}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	})
	h := p.Handle(next, map[string]any{"secret": secret, "header": "Authorization"})

	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "u", "exp": time.Now().Add(time.Hour).Unix(),
	})
	ss, _ := tok.SignedString([]byte(secret))

	req := httptest.NewRequest("GET", "/secure/x", nil)
	req.Header.Set("Authorization", "Bearer "+ss)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 204 {
		t.Fatalf("valid token: %d", rec.Code)
	}

	req = httptest.NewRequest("GET", "/secure/x", nil)
	req.Header.Set("Authorization", "Bearer fake")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Fatalf("fake token: %d", rec.Code)
	}

	expired := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "u", "exp": time.Now().Add(-time.Hour).Unix(),
	})
	es, _ := expired.SignedString([]byte(secret))
	req = httptest.NewRequest("GET", "/secure/x", nil)
	req.Header.Set("Authorization", "Bearer "+es)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Fatalf("expired: %d", rec.Code)
	}
}

func TestJWTSkipPath(t *testing.T) {
	p := &JWTPlugin{}
	h := p.Handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }),
		map[string]any{"secret": "x", "skip_paths": []any{"/health"}})
	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 204 {
		t.Fatalf("skip path got %d", rec.Code)
	}
}
