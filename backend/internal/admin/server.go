package admin

import (
	"net/http"

	"minigate/internal/balancer"
	"minigate/internal/circuit"
	"minigate/internal/config"
	"minigate/internal/metrics"
	"minigate/internal/model"
	"minigate/internal/router"
	"minigate/internal/timeutil"
)

type Server struct {
	Table    *router.Table
	LB       *balancer.Registry
	Metric   *metrics.Collector
	Reloader *config.Reloader
	Circuit  *circuit.Registry
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", s.health)
	mux.HandleFunc("GET /api/v1/stats", s.stats)
	mux.HandleFunc("GET /api/v1/routes", s.listRoutes)
	mux.HandleFunc("GET /api/v1/routes/{id}", s.getRoute)
	mux.HandleFunc("POST /api/v1/routes", s.createRoute)
	mux.HandleFunc("PUT /api/v1/routes/{id}", s.updateRoute)
	mux.HandleFunc("PATCH /api/v1/routes/{id}/toggle", s.toggleRoute)
	mux.HandleFunc("DELETE /api/v1/routes/{id}", s.deleteRoute)
	mux.HandleFunc("GET /api/v1/upstreams", s.listUpstreams)
	mux.HandleFunc("GET /api/v1/upstreams/{id}", s.getUpstream)
	mux.HandleFunc("POST /api/v1/upstreams", s.createUpstream)
	mux.HandleFunc("PUT /api/v1/upstreams/{id}", s.updateUpstream)
	mux.HandleFunc("DELETE /api/v1/upstreams/{id}", s.deleteUpstream)
	mux.HandleFunc("GET /api/v1/middlewares", s.listMW)
	mux.HandleFunc("PUT /api/v1/middlewares/{name}", s.updateMW)
	mux.HandleFunc("GET /api/v1/config", s.getConfig)
	mux.HandleFunc("GET /api/v1/config/status", s.configStatus)
	mux.HandleFunc("POST /api/v1/tokens/demo", s.demoToken)
	return withCORS(mux)
}

func (s *Server) current() *model.GatewayConfig {
	cfg := s.Table.Config()
	return &cfg
}

func (s *Server) persist(cfg *model.GatewayConfig) error {
	return s.Reloader.SaveAndApply(cfg)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeOK(w, map[string]string{"status": "up", "time": timeutil.Format(timeutil.Now())})
}

func (s *Server) stats(w http.ResponseWriter, _ *http.Request) {
	st := model.Stats{
		QPS:           s.Metric.QPS(),
		TotalRequests: s.Metric.Total(),
		ActiveRoutes:  s.Table.ActiveRouteCount(),
		Upstreams:     s.LB.Status(),
		RecentErrors:  s.Metric.Errors(),
		HotReload:     s.Reloader.Status(),
	}
	if s.Circuit != nil {
		st.Circuits = s.Circuit.Status()
	}
	writeOK(w, st)
}
