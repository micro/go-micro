package subdomain

import (
	"net"
	"net/http"
	"strings"

	"github.com/micro/go-micro/v2/api/resolver"
	path "github.com/micro/go-micro/v2/api/resolver/vpath"
	"github.com/micro/go-micro/v2/logger"
	"golang.org/x/net/publicsuffix"
)

func NewResolver(parent resolver.Resolver, opts ...resolver.Option) resolver.Resolver {
	if parent == nil {
		parent = path.NewResolver(opts...)
	}
	return &Resolver{parent}
}

type Resolver struct {
	resolver.Resolver
}

func (r *Resolver) Resolve(req *http.Request, opts ...resolver.ResolveOption) (*resolver.Endpoint, error) {
	// resolve the endpoint using path based routing
	endpoint, err := r.Resolver.Resolve(req, resolver.WithNetwork(r.ResolveNetwork(req)))
	if err != nil {
		return nil, err
	}

	return endpoint, nil
}

func (r *Resolver) ResolveNetwork(req *http.Request) string {
	// determine the host, e.g. foobar.m3o.app
	host := req.URL.Hostname()
	if len(host) == 0 {
		if h, _, err := net.SplitHostPort(req.Host); err == nil {
			host = h // host does contain a port
		} else if strings.Contains(err.Error(), "missing port in address") {
			host = req.Host // host does not contain a port
		}
	}

	// check for an ip address
	if net.ParseIP(host) != nil {
		return "micro"
	}

	// check for dev enviroment
	if host == "localhost" || host == "127.0.0.1" {
		return "micro"
	}

	// extract the top level domain plus one (e.g. 'myapp.com')
	domain, err := publicsuffix.EffectiveTLDPlusOne(host)
	if err != nil {
		logger.Debugf("Unable to extract domain from %v", host)
		return "micro"
	}

	// there was no subdomain
	if host == domain {
		return "micro"
	}

	// remove the domain from the host, leaving the subdomain, e.g. "staging.foo.myapp.com" => "staging.foo"
	subdomain := strings.TrimSuffix(host, "."+domain)

	// return the reversed subdomain as the namespace, e.g. "staging.foo" => "foo-staging"
	comps := strings.Split(subdomain, ".")
	for i := len(comps)/2 - 1; i >= 0; i-- {
		opp := len(comps) - 1 - i
		comps[i], comps[opp] = comps[opp], comps[i]
	}
	return strings.Join(comps, "-")
}

func (r *Resolver) String() string {
	return "subdomain"
}
