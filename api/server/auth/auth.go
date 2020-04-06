package auth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/micro/go-micro/v2/api/resolver"
	"github.com/micro/go-micro/v2/api/resolver/path"
	"github.com/micro/go-micro/v2/auth"
	"github.com/micro/go-micro/v2/logger"
)

// CombinedAuthHandler wraps a server and authenticates requests
func CombinedAuthHandler(namespace string, r resolver.Resolver, h http.Handler) http.Handler {
	if r == nil {
		r = path.NewResolver()
	}
	if len(namespace) == 0 {
		namespace = "go.micro"
	}

	return authHandler{
		handler:   h,
		resolver:  r,
		auth:      auth.DefaultAuth,
		namespace: namespace,
	}
}

type authHandler struct {
	handler   http.Handler
	auth      auth.Auth
	resolver  resolver.Resolver
	namespace string
}

func (h authHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Determine the namespace
	namespace, err := namespaceFromRequest(req)
	if err != nil {
		logger.Error(err)
		namespace = auth.DefaultNamespace
	}

	// Set the namespace in the header
	req.Header.Set(auth.NamespaceKey, namespace)

	// Extract the token from the request
	var token string
	if header := req.Header.Get("Authorization"); len(header) > 0 {
		// Extract the auth token from the request
		if strings.HasPrefix(header, auth.BearerScheme) {
			token = header[len(auth.BearerScheme):]
		}
	} else {
		// Get the token out the cookies if not provided in headers
		if c, err := req.Cookie("micro-token"); err == nil && c != nil {
			token = strings.TrimPrefix(c.Value, auth.TokenCookieName+"=")
			req.Header.Set("Authorization", auth.BearerScheme+token)
		}
	}

	// Get the account using the token, fallback to a blank account
	// since some endpoints can be unauthenticated, so the lack of an
	// account doesn't necesserially mean a forbidden request
	acc, err := h.auth.Inspect(token)
	if err != nil {
		acc = &auth.Account{Namespace: namespace}
	}

	// Check the accounts namespace matches the namespace we're operating
	// within. If not forbid the request and log the occurance.
	if acc.Namespace != namespace {
		logger.Warnf("Cross namespace request forbidden: account %v (%v) requested access to %v in the %v namespace", acc.ID, acc.Namespace, req.URL.Path, namespace)
		http.Error(w, "Forbidden namespace", 403)
	}

	// Determine the name of the service being requested
	endpoint, err := h.resolver.Resolve(req)
	if err == resolver.ErrInvalidPath || err == resolver.ErrNotFound {
		// a file not served by the resolver has been requested (e.g. favicon.ico)
		endpoint = &resolver.Endpoint{Path: req.URL.Path}
	} else if err != nil {
		logger.Error(err)
		http.Error(w, err.Error(), 500)
		return
	} else {
		// set the endpoint in the context so it can be used to resolve
		// the request later
		ctx := context.WithValue(req.Context(), resolver.Endpoint{}, endpoint)
		*req = *req.Clone(ctx)
	}

	// construct the resource name, e.g. home => go.micro.web.home
	resName := h.namespace
	if len(endpoint.Name) > 0 {
		resName = resName + "." + endpoint.Name
	}

	// determine the resource path. there is an inconsistency in how resolvers
	// use method, some use it as Users.ReadUser (the rpc method), and others
	// use it as the HTTP method, e.g GET. TODO: Refactor this to make it consistent.
	resEndpoint := endpoint.Path
	if len(endpoint.Path) == 0 {
		resEndpoint = endpoint.Method
	}

	// Perform the verification check to see if the account has access to
	// the resource they're requesting
	res := &auth.Resource{Type: "service", Name: resName, Endpoint: resEndpoint, Namespace: namespace}
	if err := h.auth.Verify(acc, res); err == nil {
		// The account has the necessary permissions to access the resource
		h.handler.ServeHTTP(w, req)
		return
	}

	// The account is set, but they don't have enough permissions, hence
	// we return a forbidden error.
	if len(acc.ID) > 0 {
		http.Error(w, "Forbidden request", 403)
		return
	}

	// If there is no auth login url set, 401
	loginURL := h.auth.Options().LoginURL
	if loginURL == "" {
		http.Error(w, "unauthorized request", 401)
		return
	}

	// Redirect to the login path
	params := url.Values{"redirect_to": {req.URL.Path}}
	loginWithRedirect := fmt.Sprintf("%v?%v", loginURL, params.Encode())
	http.Redirect(w, req, loginWithRedirect, http.StatusTemporaryRedirect)
}

func namespaceFromRequest(req *http.Request) (string, error) {
	// needed to tmp debug host in prod. will be removed.
	logger.Infof("Host is '%v'; URL Host is '%v'; URL Hostname is '%v'", req.Host, req.URL.Host, req.URL.Hostname())

	// determine the host, e.g. dev.micro.mu:8080
	host := req.URL.Hostname()
	if len(host) == 0 {
		// fallback to req.Host
		var err error
		host, _, err = net.SplitHostPort(req.Host)
		if err != nil && err.Error() == "missing port in address" {
			host = req.Host
		}
	}

	// check for an ip address
	if net.ParseIP(host) != nil {
		return auth.DefaultNamespace, nil
	}

	// check for dev enviroment
	if host == "localhost" || host == "127.0.0.1" {
		return auth.DefaultNamespace, nil
	}

	// TODO: this logic needs to be replaced with usage of publicsuffix
	// if host is not a subdomain, deturn default namespace
	comps := strings.Split(host, ".")
	if len(comps) != 3 {
		return auth.DefaultNamespace, nil
	}

	// check for the micro.mu domain
	domain := fmt.Sprintf("%v.%v", comps[1], comps[2])
	if domain == "micro.mu" {
		return auth.DefaultNamespace, nil
	}

	// return the subdomain as the host
	return comps[0], nil
}
