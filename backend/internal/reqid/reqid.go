package reqid

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

type ctxKey struct{}

const Header = "X-Request-ID"

func New() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "mg-" + hex.EncodeToString(b[:8])
	}
	return hex.EncodeToString(b[:])
}

func FromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKey{}).(string); ok {
		return v
	}
	return ""
}

func With(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

func Ensure(r *http.Request) (string, *http.Request) {
	id := stringsTrim(r.Header.Get(Header))
	if id == "" {
		id = New()
	}
	r.Header.Set(Header, id)
	return id, r.WithContext(With(r.Context(), id))
}

func stringsTrim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}
