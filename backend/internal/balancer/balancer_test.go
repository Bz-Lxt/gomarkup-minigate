package balancer

import (
	"testing"

	"minigate/internal/model"
)

func spec(algo string, weights ...int) model.UpstreamSpec {
	nodes := make([]model.NodeSpec, 0, len(weights))
	for i, w := range weights {
		nodes = append(nodes, model.NodeSpec{Target: string(rune('a'+i)) + ".local", Weight: w})
	}
	return model.UpstreamSpec{ID: "u", Algorithm: algo, Nodes: nodes, FailThreshold: 3}
}

func TestRoundRobinEven(t *testing.T) {
	b := New(spec("round_robin", 1, 1, 1))
	cnt := map[string]int{}
	for i := 0; i < 999; i++ {
		n := b.Next()
		if n == nil {
			t.Fatal("nil node")
		}
		cnt[n.Target]++
	}
	for _, v := range cnt {
		if v != 333 {
			t.Fatalf("rr distribution %v", cnt)
		}
	}
}

func TestWeightedRR(t *testing.T) {
	b := New(spec("weighted_rr", 1, 3))
	cnt := map[string]int{}
	for i := 0; i < 1000; i++ {
		cnt[b.Next().Target]++
	}
	a := cnt["a.local"]
	c := cnt["b.local"]
	if a+c != 1000 {
		t.Fatalf("total %d", a+c)
	}
	ratio := float64(c) / float64(a)
	if ratio < 2.7 || ratio > 3.3 {
		t.Fatalf("weighted ratio want ~3 got %v (%d:%d)", ratio, a, c)
	}
}

func TestSkipUnhealthy(t *testing.T) {
	b := New(spec("round_robin", 1, 1))
	b.Nodes()[0].ForceDown(true)
	for i := 0; i < 20; i++ {
		n := b.Next()
		if n.Target != "b.local" {
			t.Fatalf("got unhealthy %s", n.Target)
		}
	}
}

func TestLeastConnPrefersIdle(t *testing.T) {
	b := New(spec("least_conn", 1, 1))
	b.Nodes()[0].Acquire()
	b.Nodes()[0].Acquire()
	for i := 0; i < 10; i++ {
		n := b.Next()
		if n.Target != "b.local" {
			t.Fatalf("want idle node, got %s inflight=%d", n.Target, n.Inflight())
		}
	}
}

func TestPassiveFailThreshold(t *testing.T) {
	n := &Node{Target: "x"}
	n.Report(false, 3)
	n.Report(false, 3)
	if !n.Healthy() {
		t.Fatal("should still be up")
	}
	n.Report(false, 3)
	if n.Healthy() {
		t.Fatal("should be down")
	}
	n.Report(true, 3)
	if !n.Healthy() {
		t.Fatal("should recover")
	}
}
