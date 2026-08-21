package balancer

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"minigate/internal/model"
)

type Node struct {
	Target   string
	Weight   int
	fails    atomic.Int32
	down     atomic.Bool
	inflight atomic.Int32
}

func (n *Node) Healthy() bool {
	return !n.down.Load()
}

func (n *Node) Fails() int32 {
	return n.fails.Load()
}

func (n *Node) Report(ok bool, threshold int) {
	if threshold <= 0 {
		threshold = 3
	}
	if ok {
		n.fails.Store(0)
		n.down.Store(false)
		return
	}
	if n.fails.Add(1) >= int32(threshold) {
		n.down.Store(true)
	}
}

func (n *Node) ForceDown(down bool) {
	n.down.Store(down)
	if !down {
		n.fails.Store(0)
	}
}

func (n *Node) Acquire() { n.inflight.Add(1) }

func (n *Node) Release() {
	if n.inflight.Add(-1) < 0 {
		n.inflight.Store(0)
	}
}

func (n *Node) Inflight() int32 { return n.inflight.Load() }

type Balancer interface {
	Next() *Node
	Nodes() []*Node
	Algorithm() string
}

type roundRobin struct {
	nodes []*Node
	idx   atomic.Uint64
}

func (r *roundRobin) Algorithm() string { return "round_robin" }
func (r *roundRobin) Nodes() []*Node    { return r.nodes }
func (r *roundRobin) Next() *Node {
	n := len(r.nodes)
	if n == 0 {
		return nil
	}
	for i := 0; i < n; i++ {
		idx := r.idx.Add(1) - 1
		node := r.nodes[int(idx%uint64(n))]
		if node.Healthy() {
			return node
		}
	}
	return nil
}

type randomLB struct {
	nodes []*Node
	mu    sync.Mutex
	rnd   *rand.Rand
}

func (r *randomLB) Algorithm() string { return "random" }
func (r *randomLB) Nodes() []*Node    { return r.nodes }
func (r *randomLB) Next() *Node {
	healthy := healthyOf(r.nodes)
	if len(healthy) == 0 {
		return nil
	}
	r.mu.Lock()
	i := r.rnd.Intn(len(healthy))
	r.mu.Unlock()
	return healthy[i]
}

type wnode struct {
	node    *Node
	weight  int
	current int
}

type weightedRR struct {
	mu    sync.Mutex
	nodes []*wnode
	raw   []*Node
}

func (w *weightedRR) Algorithm() string { return "weighted_rr" }
func (w *weightedRR) Nodes() []*Node    { return w.raw }
func (w *weightedRR) Next() *Node {
	w.mu.Lock()
	defer w.mu.Unlock()
	total := 0
	var best *wnode
	for _, n := range w.nodes {
		if !n.node.Healthy() {
			continue
		}
		n.current += n.weight
		total += n.weight
		if best == nil || n.current > best.current {
			best = n
		}
	}
	if best == nil {
		return nil
	}
	best.current -= total
	return best.node
}

func New(spec model.UpstreamSpec) Balancer {
	nodes := make([]*Node, 0, len(spec.Nodes))
	for _, ns := range spec.Nodes {
		w := ns.Weight
		if w <= 0 {
			w = 1
		}
		nodes = append(nodes, &Node{Target: ns.Target, Weight: w})
	}
	switch spec.Algorithm {
	case "random":
		return &randomLB{nodes: nodes, rnd: rand.New(rand.NewSource(time.Now().UnixNano()))}
	case "weighted_rr":
		wn := make([]*wnode, 0, len(nodes))
		for _, n := range nodes {
			wn = append(wn, &wnode{node: n, weight: n.Weight})
		}
		return &weightedRR{nodes: wn, raw: nodes}
	case "least_conn":
		return newLeastConn(nodes)
	default:
		return &roundRobin{nodes: nodes}
	}
}

func healthyOf(nodes []*Node) []*Node {
	out := make([]*Node, 0, len(nodes))
	for _, n := range nodes {
		if n.Healthy() {
			out = append(out, n)
		}
	}
	return out
}

type Registry struct {
	mu   sync.RWMutex
	data map[string]Balancer
	meta map[string]model.UpstreamSpec
}

func NewRegistry() *Registry {
	return &Registry{data: map[string]Balancer{}, meta: map[string]model.UpstreamSpec{}}
}

func (r *Registry) Rebuild(upstreams []model.UpstreamSpec) {
	next := make(map[string]Balancer, len(upstreams))
	meta := make(map[string]model.UpstreamSpec, len(upstreams))
	for _, u := range upstreams {
		next[u.ID] = New(u)
		meta[u.ID] = u
	}
	r.mu.Lock()
	r.data = next
	r.meta = meta
	r.mu.Unlock()
}

func (r *Registry) Get(id string) (Balancer, model.UpstreamSpec, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, ok := r.data[id]
	if !ok {
		return nil, model.UpstreamSpec{}, false
	}
	return b, r.meta[id], true
}

func (r *Registry) Status() []model.UpstreamStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]model.UpstreamStatus, 0, len(r.data))
	for id, b := range r.data {
		meta := r.meta[id]
		st := model.UpstreamStatus{
			ID:        id,
			Name:      meta.Name,
			Algorithm: b.Algorithm(),
		}
		for _, n := range b.Nodes() {
			ok := n.Healthy()
			if ok {
				st.Healthy++
			}
			st.Total++
			st.Nodes = append(st.Nodes, model.NodeRuntime{
				Target:  n.Target,
				Weight:  n.Weight,
				Healthy: ok,
				Fails:   n.Fails(),
			})
		}
		out = append(out, st)
	}
	return out
}

func (r *Registry) All() map[string]Balancer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cp := make(map[string]Balancer, len(r.data))
	for k, v := range r.data {
		cp[k] = v
	}
	return cp
}
