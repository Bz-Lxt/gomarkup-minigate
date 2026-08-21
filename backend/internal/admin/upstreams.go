package admin

import (
	"net/http"

	"minigate/internal/config"
	"minigate/internal/model"
)

func (s *Server) listUpstreams(w http.ResponseWriter, _ *http.Request) {
	writeOK(w, s.Table.Upstreams())
}

func (s *Server) getUpstream(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if u, ok := s.Table.Upstream(id); ok {
		writeOK(w, u)
		return
	}
	writeErr(w, 404, 40401, "upstream not found")
}

func (s *Server) createUpstream(w http.ResponseWriter, r *http.Request) {
	var body model.UpstreamSpec
	if err := decode(r, &body); err != nil {
		writeErr(w, 400, 40001, "invalid json: "+err.Error())
		return
	}
	cfg := config.Clone(s.current())
	for _, u := range cfg.Upstreams {
		if u.ID == body.ID {
			writeErr(w, 409, 40901, "upstream id already exists")
			return
		}
	}
	cfg.Upstreams = append(cfg.Upstreams, body)
	if err := s.persist(cfg); err != nil {
		writeErr(w, 400, 40001, err.Error())
		return
	}
	writeJSON(w, 201, 0, "ok", body)
}

func (s *Server) updateUpstream(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body model.UpstreamSpec
	if err := decode(r, &body); err != nil {
		writeErr(w, 400, 40001, "invalid json: "+err.Error())
		return
	}
	body.ID = id
	cfg := config.Clone(s.current())
	found := false
	for i, u := range cfg.Upstreams {
		if u.ID == id {
			cfg.Upstreams[i] = body
			found = true
			break
		}
	}
	if !found {
		writeErr(w, 404, 40401, "upstream not found")
		return
	}
	if err := s.persist(cfg); err != nil {
		writeErr(w, 400, 40001, err.Error())
		return
	}
	writeOK(w, body)
}

func (s *Server) deleteUpstream(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cfg := config.Clone(s.current())
	for _, rt := range cfg.Routes {
		if rt.UpstreamID == id {
			writeErr(w, 400, 40001, "upstream is referenced by route "+rt.ID)
			return
		}
	}
	out := cfg.Upstreams[:0]
	found := false
	for _, u := range cfg.Upstreams {
		if u.ID == id {
			found = true
			continue
		}
		out = append(out, u)
	}
	if !found {
		writeErr(w, 404, 40401, "upstream not found")
		return
	}
	cfg.Upstreams = out
	if err := s.persist(cfg); err != nil {
		writeErr(w, 400, 40001, err.Error())
		return
	}
	writeOK(w, nil)
}
