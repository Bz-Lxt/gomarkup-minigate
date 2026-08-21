package admin

import (
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"minigate/internal/config"
	"minigate/internal/middleware"
	"minigate/internal/model"
	"minigate/internal/timeutil"
)

func pluginDesc() map[string]string {
	return map[string]string{
		"jwt":       "HS256 认证，可配 secret / header / 白名单路径",
		"ratelimit": "令牌桶 / 漏桶限流，维度 global / route / ip",
		"logger":    "访问日志（方法、路径、状态、耗时、上游、IP、request_id）",
		"cors":      "跨域预检与 Origin 白名单",
		"rewrite":   "路径改写：strip_prefix / add_prefix / set_path",
		"headers":   "请求/响应头注入",
		"ipfilter":  "IP 允许/拒绝列表，支持 CIDR",
	}
}

func (s *Server) listMW(w http.ResponseWriter, _ *http.Request) {
	cfg := s.current()
	type item struct {
		Name        string         `json:"name"`
		Enabled     bool           `json:"enabled"`
		Scope       string         `json:"scope"`
		Config      map[string]any `json:"config"`
		Registered  bool           `json:"registered"`
		Description string         `json:"description"`
	}
	desc := pluginDesc()
	seen := map[string]bool{}
	out := make([]item, 0)
	for _, m := range cfg.GlobalMiddlewares {
		out = append(out, item{
			Name: m.Name, Enabled: m.Enabled, Scope: "global", Config: m.Config,
			Registered: true, Description: desc[m.Name],
		})
		seen[m.Name] = true
	}
	for _, name := range middleware.Names() {
		if seen[name] {
			continue
		}
		out = append(out, item{
			Name: name, Enabled: false, Scope: "global", Config: map[string]any{},
			Registered: true, Description: desc[name],
		})
	}
	writeOK(w, out)
}

func (s *Server) updateMW(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var body model.MiddlewareSpec
	if err := decode(r, &body); err != nil {
		writeErr(w, 400, 40001, "invalid json: "+err.Error())
		return
	}
	body.Name = name
	if err := middleware.ValidateNamed(name, body.Config); err != nil {
		writeErr(w, 400, 40001, err.Error())
		return
	}
	cfg := config.Clone(s.current())
	found := false
	for i, m := range cfg.GlobalMiddlewares {
		if m.Name == name {
			cfg.GlobalMiddlewares[i] = body
			found = true
			break
		}
	}
	if !found {
		cfg.GlobalMiddlewares = append(cfg.GlobalMiddlewares, body)
	}
	if err := s.persist(cfg); err != nil {
		writeErr(w, 500, 50002, "middleware not applied: "+err.Error())
		return
	}
	writeOK(w, body)
}

func (s *Server) getConfig(w http.ResponseWriter, _ *http.Request) {
	writeOK(w, s.current())
}

func (s *Server) configStatus(w http.ResponseWriter, _ *http.Request) {
	writeOK(w, s.Reloader.Status())
}

func (s *Server) demoToken(w http.ResponseWriter, _ *http.Request) {
	secret := "minigate-dev-secret"
	for _, m := range s.Table.Globals() {
		if m.Name == "jwt" && m.Config != nil {
			if v, ok := m.Config["secret"]; ok && v != nil {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					secret = s
				}
			}
		}
	}
	exp := timeutil.Now().Add(time.Hour)
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "demo",
		"exp": exp.Unix(),
		"iat": timeutil.Now().Unix(),
	})
	ss, err := tok.SignedString([]byte(secret))
	if err != nil {
		writeErr(w, 500, 50001, err.Error())
		return
	}
	writeOK(w, map[string]string{"token": ss, "expires_at": timeutil.Format(exp)})
}
