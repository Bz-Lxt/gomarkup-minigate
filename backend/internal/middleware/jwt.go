package middleware

import (
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

func init() {
	Register(&JWTPlugin{})
}

type JWTPlugin struct{}

func (p *JWTPlugin) Name() string { return "jwt" }

func (p *JWTPlugin) ValidateConfig(cfg map[string]any) error {
	if strVal(cfg, "secret", "") == "" {
		return nil
	}
	return nil
}

func (p *JWTPlugin) Handle(next http.Handler, cfg map[string]any) http.Handler {
	secret := strVal(cfg, "secret", "minigate-dev-secret")
	header := strVal(cfg, "header", "Authorization")
	skips := strSlice(cfg, "skip_paths")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, s := range skips {
			if s != "" && (r.URL.Path == s || strings.HasPrefix(r.URL.Path, s+"/")) {
				next.ServeHTTP(w, r)
				return
			}
		}
		raw := r.Header.Get(header)
		if strings.HasPrefix(strings.ToLower(raw), "bearer ") {
			raw = strings.TrimSpace(raw[7:])
		}
		if raw == "" {
			http.Error(w, `{"code":40101,"message":"missing token"}`, http.StatusUnauthorized)
			return
		}
		tok, err := jwt.Parse(raw, func(t *jwt.Token) (any, error) {
			if t.Method != jwt.SigningMethodHS256 {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
		if err != nil || !tok.Valid {
			http.Error(w, `{"code":40102,"message":"invalid or expired token"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
