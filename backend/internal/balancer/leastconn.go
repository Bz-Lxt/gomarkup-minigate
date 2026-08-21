package balancer

import "sync"

type leastConn struct {
	mu    sync.Mutex
	nodes []*Node
}

func (l *leastConn) Algorithm() string { return "least_conn" }
func (l *leastConn) Nodes() []*Node    { return l.nodes }

func (l *leastConn) Next() *Node {
	l.mu.Lock()
	defer l.mu.Unlock()
	var best *Node
	for _, n := range l.nodes {
		if !n.Healthy() {
			continue
		}
		if best == nil || n.Inflight() < best.Inflight() {
			best = n
		}
	}
	return best
}

func newLeastConn(nodes []*Node) *leastConn {
	return &leastConn{nodes: nodes}
}
