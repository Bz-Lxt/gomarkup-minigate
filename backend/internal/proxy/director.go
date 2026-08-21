package proxy

import (
	"net/http"
	"net/url"
	"strings"

	"minigate/internal/model"
)

func rewritePath(path, stripPrefix, targetPath string) string {
	if stripPrefix != "" && strings.HasPrefix(path, stripPrefix) {
		rest := path[len(stripPrefix):]
		// Only strip at a segment boundary so that "/echoextra" is not
		// mistaken for a child of the "/echo" prefix.
		if rest == "" || strings.HasPrefix(rest, "/") {
			path = rest
			if path == "" {
				path = "/"
			}
		}
	}
	base := strings.TrimRight(targetPath, "/")
	if base != "" && base != "/" {
		path = base + path
	}
	return path
}

func applyDirector(req *http.Request, target *url.URL, route *model.RouteSpec, params map[string]string) {
	req.URL.Scheme = target.Scheme
	req.URL.Host = target.Host
	req.Host = target.Host
	req.URL.Path = rewritePath(req.URL.Path, route.StripPrefix, target.Path)
	req.Header.Set("X-Minigate-Route", route.ID)
	if v, ok := params["id"]; ok {
		req.Header.Set("X-Minigate-Param-Id", v)
	}
}
