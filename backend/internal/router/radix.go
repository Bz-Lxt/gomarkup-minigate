package router

import (
	"strings"

	"minigate/internal/model"
)

const (
	ntStatic uint8 = iota
	ntParam
	ntCatchAll
)

type leaf struct {
	byMethod map[string][]*model.RouteSpec
	any      []*model.RouteSpec
}

type node struct {
	prefix   string
	nType    uint8
	param    string
	children []*node
	leaf     *leaf
}

func newNode(prefix string, t uint8) *node {
	return &node{prefix: prefix, nType: t}
}

func commonPrefix(a, b string) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	i := 0
	for i < n && a[i] == b[i] {
		i++
	}
	return i
}

func splitSegments(path string) []seg {
	if path == "" {
		return nil
	}
	out := make([]seg, 0, 8)
	i := 0
	for i < len(path) {
		if path[i] == '{' {
			j := strings.IndexByte(path[i:], '}')
			if j < 0 {
				out = append(out, seg{ntStatic, path[i:]})
				break
			}
			name := path[i+1 : i+j]
			out = append(out, seg{ntParam, name})
			i = i + j + 1
			continue
		}
		if path[i] == '*' {
			out = append(out, seg{ntCatchAll, "*"})
			break
		}
		j := i + 1
		for j < len(path) && path[j] != '{' && path[j] != '*' {
			j++
		}
		out = append(out, seg{ntStatic, path[i:j]})
		i = j
	}
	return out
}

type seg struct {
	t    uint8
	text string
}

func (n *node) insert(path string, methods []string, route *model.RouteSpec) {
	segs := splitSegments(path)
	cur := n
	for _, s := range segs {
		cur = cur.childFor(s)
	}
	if cur.leaf == nil {
		cur.leaf = &leaf{byMethod: map[string][]*model.RouteSpec{}}
	}
	if len(methods) == 0 {
		cur.leaf.any = append(cur.leaf.any, route)
		return
	}
	for _, m := range methods {
		m = strings.ToUpper(strings.TrimSpace(m))
		if m == "" {
			continue
		}
		cur.leaf.byMethod[m] = append(cur.leaf.byMethod[m], route)
	}
}

func (n *node) childFor(s seg) *node {
	if s.t != ntStatic {
		for _, c := range n.children {
			if c.nType == s.t && c.param == s.text {
				return c
			}
		}
		ch := newNode("", s.t)
		ch.param = s.text
		n.children = append(n.children, ch)
		return ch
	}
	for i, c := range n.children {
		if c.nType != ntStatic {
			continue
		}
		cp := commonPrefix(c.prefix, s.text)
		if cp == 0 {
			continue
		}
		if cp == len(c.prefix) {
			if cp == len(s.text) {
				return c
			}
			return c.childFor(seg{ntStatic, s.text[cp:]})
		}
		split := newNode(c.prefix[:cp], ntStatic)
		c.prefix = c.prefix[cp:]
		split.children = []*node{c}
		n.children[i] = split
		if cp == len(s.text) {
			return split
		}
		return split.childFor(seg{ntStatic, s.text[cp:]})
	}
	ch := newNode(s.text, ntStatic)
	n.children = append(n.children, ch)
	return ch
}

type Match struct {
	Route  *model.RouteSpec
	Params map[string]string
}

func better(a, b *model.RouteSpec) *model.RouteSpec {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	if b.Priority > a.Priority {
		return b
	}
	return a
}

func pickLeaf(l *leaf, method, host string) *model.RouteSpec {
	if l == nil {
		return nil
	}
	var best *model.RouteSpec
	for _, r := range l.byMethod[method] {
		if r.Enabled && hostOK(r, host) {
			best = better(best, r)
		}
	}
	for _, r := range l.any {
		if r.Enabled && hostOK(r, host) {
			best = better(best, r)
		}
	}
	return best
}

func hostOK(r *model.RouteSpec, host string) bool {
	if r.Host == "" {
		return true
	}
	h := host
	if i := strings.IndexByte(h, ':'); i >= 0 {
		h = h[:i]
	}
	return strings.EqualFold(r.Host, h)
}

func (n *node) match(method, path, host string) (*model.RouteSpec, map[string]string) {
	params := map[string]string{}
	r := n.walk(method, path, host, params)
	if r == nil {
		return nil, nil
	}
	return r, params
}

func (n *node) walk(method, path, host string, params map[string]string) *model.RouteSpec {
	switch n.nType {
	case ntCatchAll:
		params["*"] = path
		return pickLeaf(n.leaf, method, host)
	case ntParam:
		end := strings.IndexByte(path, '/')
		var val, rest string
		if end < 0 {
			val = path
			rest = ""
		} else {
			val = path[:end]
			rest = path[end:]
		}
		params[n.param] = val
		var best *model.RouteSpec
		if rest == "" {
			best = pickLeaf(n.leaf, method, host)
		}
		for _, c := range n.children {
			if cand := c.walk(method, rest, host, params); cand != nil {
				best = better(best, cand)
			}
		}
		return best
	default:
		if !strings.HasPrefix(path, n.prefix) {
			// Allow the bare prefix of a catch-all pattern to match even
			// when the request omits the trailing slash. E.g. "/echo"
			// should still hit a "/echo/*" route so that strip_prefix and
			// the rewrite logic run for the root address.
			if strings.HasSuffix(n.prefix, "/") && path+"/" == n.prefix && len(n.children) > 0 {
				for _, c := range n.children {
					if c.nType == ntCatchAll {
						if c.leaf != nil {
							if r := pickLeaf(c.leaf, method, host); r != nil {
								params["*"] = ""
								return r
							}
						}
					}
				}
			}
			return nil
		}
		rest := path[len(n.prefix):]
		var best *model.RouteSpec
		if rest == "" {
			best = pickLeaf(n.leaf, method, host)
		}
		for _, c := range n.children {
			if cand := c.walk(method, rest, host, params); cand != nil {
				best = better(best, cand)
			}
		}
		return best
	}
}
