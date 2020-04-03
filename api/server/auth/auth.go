package auth

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/micro/go-micro/v2/api/resolver"
	"github.com/micro/go-micro/v2/api/resolver/path"
	"github.com/micro/go-micro/v2/auth"
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
		acc = &auth.Account{}
	}

	// Determine the name of the service being requested
	endpoint, err := h.resolver.Resolve(req)
	if err == resolver.ErrInvalidPath || err == resolver.ErrNotFound {
		// a file not served by the resolver has been requested (e.g. favicon.ico)
		endpoint = &resolver.Endpoint{Path: req.URL.Path}
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
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
	res := &auth.Resource{Type: "service", Name: resName, Endpoint: resEndpoint}
	if err := h.auth.Verify(acc, res); err == nil {
		// The account has the necessary permissions to access the resource
		h.handler.ServeHTTP(w, req)
		return
	}

	// The account is set, but they don't have enough permissions,
	// hence we 403.
	if len(acc.ID) > 0 {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// If there is no auth login url set, 401
	loginURL := h.auth.Options().LoginURL
	if loginURL == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Redirect to the login path
	params := url.Values{"redirect_to": {req.URL.Path}}
	loginWithRedirect := fmt.Sprintf("%v?%v", loginURL, params.Encode())
	http.Redirect(w, req, loginWithRedirect, http.StatusTemporaryRedirect)
}
