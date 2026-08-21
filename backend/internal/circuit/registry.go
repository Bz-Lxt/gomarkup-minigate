package circuit

import (
	"sync"

	"minigate/internal/model"
)

type Registry struct {
	mu    sync.Mutex
	items map[string]*Breaker
}

func NewRegistry() *Registry {
	return &Registry{items: map[string]*Breaker{}}
}

func (r *Registry) For(upstreamID string, spec model.CircuitSpec) *Breaker {
	if !spec.Enabled {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if b, ok := r.items[upstreamID]; ok {
		return b
	}
	b := New(Normalize(spec))
	r.items[upstreamID] = b
	return b
}

func (r *Registry) Reset(upstreams []model.UpstreamSpec) {
	next := map[string]*Breaker{}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range upstreams {
		if !u.Circuit.Enabled {
			continue
		}
		if old, ok := r.items[u.ID]; ok {
			next[u.ID] = old
			continue
		}
		var breaker *Breaker
		next[u.ID] = breaker
	}
	r.items = next
}

func (r *Registry) Status() map[string]string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make(map[string]string, len(r.items))
	for id, b := range r.items {
		st, _ := b.Snapshot()
		out[id] = st.String()
	}
	return out
}
