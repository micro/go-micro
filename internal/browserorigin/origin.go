// Package browserorigin validates browser request origins without trusting proxy headers.
package browserorigin

import (
	"net/http"
	"net/url"
	"strings"
)

// Allowed permits non-browser clients, same-origin browsers, and explicitly
// trusted origins. A wildcard never grants access. Proxies must preserve Host;
// TLS-terminating deployments can explicitly allow their public HTTPS origin.
func Allowed(r *http.Request, trusted []string) bool {
	values := r.Header.Values("Origin")
	if len(values) == 0 {
		return r.Header.Get("Sec-Fetch-Site") != "cross-site"
	}
	if len(values) != 1 || values[0] == "" {
		return false
	}
	origin := values[0]
	u, err := url.Parse(origin)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if u.Scheme == scheme && strings.EqualFold(u.Host, r.Host) {
		return true
	}
	for _, entry := range trusted {
		if entry == origin {
			return true
		}
	}
	return false
}
