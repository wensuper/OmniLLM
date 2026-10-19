package relay

import (
	"net/http"
	"net/url"
	"strings"
)

// webProtocols reads a browser device's subprotocols: whether it speaks
// WebProtocol, and the secret it offered.
func webProtocols(r *http.Request) (web bool, auth string) {
	for _, h := range r.Header.Values("Sec-WebSocket-Protocol") {
		for _, p := range strings.Split(h, ",") {
			p = strings.TrimSpace(p)
			switch {
			case p == WebProtocol:
				web = true
			case strings.HasPrefix(p, WebAuthPrefix) && len(p) <= len(WebAuthPrefix)+64:
				auth = p[len(WebAuthPrefix):]
			}
		}
	}
	return web, auth
}

func (s *Server) webOrigins() []string {
	if s.opts.WebOrigins != nil {
		return s.opts.WebOrigins
	}
	return DefaultWebOrigins
}

// webOriginAllowed accepts exactly the configured origins.
func (s *Server) webOriginAllowed(origin string) bool {
	for _, o := range s.webOrigins() {
		if origin == o {
			return true
		}
	}
	return false
}

// originHosts is the same list in the form the WebSocket library checks.
func (s *Server) originHosts() []string {
	var out []string
	for _, o := range s.webOrigins() {
		if u, err := url.Parse(o); err == nil && u.Host != "" {
			out = append(out, u.Host)
		}
	}
	return out
}
