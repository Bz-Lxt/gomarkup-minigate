package proxy

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"minigate/internal/balancer"
	"minigate/internal/circuit"
	"minigate/internal/metrics"
	"minigate/internal/middleware"
	"minigate/internal/model"
	"minigate/internal/reqid"
	"minigate/internal/router"
)

type Engine struct {
	Table   *router.Table
	LB      *balancer.Registry
	Metric  *metrics.Collector
	Circuit *circuit.Registry
}

func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	e.Metric.Hit()
	id, r := reqid.Ensure(r)
	w.Header().Set(reqid.Header, id)
	route, params := e.Table.Match(r.Method, r.URL.Path, r.Host)
	if route == nil || !route.Enabled {
		http.Error(w, `{"code":40401,"message":"no matching route"}`, http.StatusNotFound)
		return
	}
	lb, spec, ok := e.LB.Get(route.UpstreamID)
	if !ok {
		e.Metric.Error("upstream not found: " + route.UpstreamID)
		http.Error(w, `{"code":50201,"message":"upstream not found"}`, http.StatusBadGateway)
		return
	}
	if e.Circuit != nil && spec.Circuit.Enabled {
		if br := e.Circuit.For(spec.ID, spec.Circuit); br != nil && !br.Allow() {
			e.Metric.Error("circuit open: " + spec.ID)
			http.Error(w, `{"code":50301,"message":"circuit breaker open"}`, http.StatusServiceUnavailable)
			return
		}
	}
	core := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		e.forward(w, r, route, spec, lb, params)
	})
	lookup := func(name string) (model.MiddlewareSpec, bool) {
		for _, m := range e.Table.Globals() {
			if m.Name == name {
				return m, true
			}
		}
		return model.MiddlewareSpec{}, false
	}
	middleware.BuildChain(core, e.Table.Globals(), route.Middlewares, lookup).ServeHTTP(w, r)
}

func (e *Engine) reportCircuit(spec model.UpstreamSpec, ok bool) {
	if e.Circuit == nil || !spec.Circuit.Enabled {
		return
	}
	br := e.Circuit.For(spec.ID, spec.Circuit)
	if br == nil {
		return
	}
	if ok {
		br.Success()
		return
	}
	br.Failure()
}

func (e *Engine) forward(w http.ResponseWriter, r *http.Request, route *model.RouteSpec, spec model.UpstreamSpec, lb balancer.Balancer, params map[string]string) {
	timeout := time.Duration(spec.TimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	try := 1
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		try = 2
	}
	var lastErr error
	for i := 0; i < try; i++ {
		node := lb.Next()
		if node == nil {
			e.Metric.Error("no healthy node for " + spec.ID)
			e.reportCircuit(spec, false)
			http.Error(w, `{"code":50202,"message":"no healthy upstream node"}`, http.StatusBadGateway)
			return
		}
		target, err := url.Parse(node.Target)
		if err != nil {
			lastErr = err
			node.Report(false, spec.FailThreshold)
			continue
		}
		last := i == try-1
		ok := e.doProxy(w, r, target, route, params, timeout, node, spec.FailThreshold, last)
		if ok {
			e.reportCircuit(spec, true)
			return
		}
		lastErr = fmt.Errorf("proxy %s failed", node.Target)
	}
	e.reportCircuit(spec, false)
	if lastErr != nil {
		e.Metric.Error(lastErr.Error())
	}
	http.Error(w, `{"code":50203,"message":"upstream request failed"}`, http.StatusBadGateway)
}

type bufferWriter struct {
	header http.Header
	code   int
	body   []byte
}

func (b *bufferWriter) Header() http.Header {
	if b.header == nil {
		b.header = make(http.Header)
	}
	return b.header
}
func (b *bufferWriter) Write(p []byte) (int, error) {
	if b.code == 0 {
		b.code = 200
	}
	b.body = append(b.body, p...)
	return len(p), nil
}
func (b *bufferWriter) WriteHeader(code int) { b.code = code }

func (e *Engine) doProxy(w http.ResponseWriter, r *http.Request, target *url.URL, route *model.RouteSpec, params map[string]string, timeout time.Duration, node *balancer.Node, threshold int, last bool) bool {
	rp := httputil.NewSingleHostReverseProxy(target)
	rp.Transport = &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		DialContext:         (&net.Dialer{Timeout: 3 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		MaxIdleConns:        256,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 5 * time.Second,
	}
	failed := false
	rp.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
		failed := err != nil
		if failed {
			node.Report(false, threshold)
		}
		e.Metric.Error(fmt.Sprintf("%s -> %s: %v", route.ID, node.Target, err))
		if last {
			http.Error(rw, `{"code":50203,"message":"upstream request failed"}`, http.StatusBadGateway)
		}
	}
	rp.Director = func(req *http.Request) {
		applyDirector(req, target, route, params)
	}
	rp.ModifyResponse = func(resp *http.Response) error {
		resp.Header.Set("X-Minigate-Upstream", node.Target)
		if resp.StatusCode >= 500 {
			failed = true
			node.Report(false, threshold)
		} else {
			node.Report(true, threshold)
		}
		return nil
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()
	req := r.WithContext(ctx)
	req.Header.Set("X-Minigate-Upstream", node.Target)
	node.Acquire()
	defer node.Release()
	if last {
		rp.ServeHTTP(w, req)
		return !failed
	}
	buf := &bufferWriter{}
	rp.ServeHTTP(buf, req)
	if failed || buf.code >= 500 {
		return false
	}
	for k, vs := range buf.header {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	if buf.code == 0 {
		buf.code = 200
	}
	w.WriteHeader(buf.code)
	_, _ = w.Write(buf.body)
	return true
}
