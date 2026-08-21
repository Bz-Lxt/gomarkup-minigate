package admin

import (
	"net/http"

	"minigate/internal/config"
	"minigate/internal/model"
	"minigate/internal/timeutil"
)

func (s *Server) listRoutes(w http.ResponseWriter, _ *http.Request) {
	writeOK(w, s.Table.Routes())
}

func (s *Server) getRoute(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	for _, rt := range s.Table.Routes() {
		if rt.ID == id {
			writeOK(w, rt)
			return
		}
	}
	writeErr(w, 404, 40401, "route not found")
}

func (s *Server) createRoute(w http.ResponseWriter, r *http.Request) {
	var body model.RouteSpec
	if err := decode(r, &body); err != nil {
		writeErr(w, 400, 40001, "invalid json: "+err.Error())
		return
	}
	now := timeutil.Format(timeutil.Now())
	body.CreatedAt = now
	body.UpdatedAt = now
	cfg := config.Clone(s.current())
	for _, rt := range cfg.Routes {
		if rt.ID == body.ID {
			writeErr(w, 409, 40901, "route id already exists")
			return
		}
	}
	cfg.Routes = append(cfg.Routes, body)
	if err := s.persist(cfg); err != nil {
		writeErr(w, 400, 40001, err.Error())
		return
	}
	writeJSON(w, 201, 0, "ok", body)
}

func (s *Server) updateRoute(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body model.RouteSpec
	if err := decode(r, &body); err != nil {
		writeErr(w, 400, 40001, "invalid json: "+err.Error())
		return
	}
	body.ID = id
	cfg := config.Clone(s.current())
	found := false
	for i, rt := range cfg.Routes {
		if rt.ID == id {
			if rt.CreatedAt != "" {
				body.CreatedAt = rt.CreatedAt
			}
			body.UpdatedAt = timeutil.Format(timeutil.Now())
			cfg.Routes[i] = body
			found = true
			break
		}
	}
	if !found {
		writeErr(w, 404, 40401, "route not found")
		return
	}
	if err := s.persist(cfg); err != nil {
		writeErr(w, 400, 40001, err.Error())
		return
	}
	writeOK(w, body)
}

func (s *Server) toggleRoute(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cfg := config.Clone(s.current())
	for i, rt := range cfg.Routes {
		if rt.ID == id {
			cfg.Routes[i].Enabled = !rt.Enabled
			cfg.Routes[i].UpdatedAt = timeutil.Format(timeutil.Now())
			if err := s.persist(cfg); err != nil {
				writeErr(w, 400, 40001, err.Error())
				return
			}
			writeOK(w, cfg.Routes[i])
			return
		}
	}
	writeErr(w, 404, 40401, "route not found")
}

func (s *Server) deleteRoute(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cfg := config.Clone(s.current())
	out := cfg.Routes[:0]
	found := false
	for _, rt := range cfg.Routes {
		if rt.ID == id {
			found = true
			continue
		}
		out = append(out, rt)
	}
	if !found {
		writeErr(w, 404, 40401, "route not found")
		return
	}
	cfg.Routes = out
	if err := s.persist(cfg); err != nil {
		writeErr(w, 400, 40001, err.Error())
		return
	}
	writeJSON(w, 200, 0, "ok", nil)
}
