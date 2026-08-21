package middleware

import (
	"bufio"
	"log/slog"
	"net"
	"net/http"
	"time"

	"minigate/internal/logger"
	"minigate/internal/netutil"
	"minigate/internal/reqid"
)

func init() {
	Register(&LoggerPlugin{})
}

type LoggerPlugin struct{}

func (p *LoggerPlugin) Name() string { return "logger" }

func (p *LoggerPlugin) ValidateConfig(map[string]any) error { return nil }

type statusWriter struct {
	http.ResponseWriter
	code int
}

func (w *statusWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := w.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

func (p *LoggerPlugin) Handle(next http.Handler, _ map[string]any) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, code: 200}
		next.ServeHTTP(sw, r)
		upstream := r.Header.Get("X-Minigate-Upstream")
		logger.L().Info("access",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", sw.code),
			slog.Int64("latency_ms", time.Since(start).Milliseconds()),
			slog.String("client_ip", netutil.ClientIP(r)),
			slog.String("upstream", upstream),
			slog.String("request_id", reqid.FromContext(r.Context())),
		)
	})
}
