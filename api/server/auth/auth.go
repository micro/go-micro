package auth

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/micro/go-micro/v2/auth"
)

var (
	// DefaultExcludes is the paths which are allowed by default
	DefaultExcludes = []string{"/favicon.ico"}
)

// CombinedAuthHandler wraps a server and authenticates requests
func CombinedAuthHandler(h http.Handler) http.Handler {
	return authHandler{
		handler: h,
		auth:    auth.DefaultAuth,
	}
}

type authHandler struct {
	handler http.Handler
	auth    auth.Auth
}

const (
	// BearerScheme is the prefix in the auth header
	BearerScheme = "Bearer "
)

func (h authHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Extract the token from the request
	var token string
	if header := req.Header.Get("Authorization"); len(header) > 0 {
		// Extract the auth token from the request
		if strings.HasPrefix(header, BearerScheme) {
			token = header[len(BearerScheme):]
		}
	} else {
		// Get the token out the cookies if not provided in headers
		if c, err := req.Cookie("micro-token"); err == nil && c != nil {
			token = strings.TrimPrefix(c.Value, auth.TokenCookieName+"=")
			req.Header.Set("Authorization", BearerScheme+token)
		}
	}

	// Return if the user disabled auth on this endpoint
	excludes := h.auth.Options().Exclude
	excludes = append(excludes, DefaultExcludes...)

	loginURL := h.auth.Options().LoginURL
	if len(loginURL) > 0 {
		excludes = append(excludes, loginURL)
	}

	for _, e := range excludes {
		// is a standard exclude, e.g. /rpc
		if e == req.URL.Path {
			h.handler.ServeHTTP(w, req)
			return
		}

		// is a wildcard exclude, e.g. /services/*
		wildcard := strings.Replace(e, "*", "", 1)
		if strings.HasSuffix(e, "*") && strings.HasPrefix(req.URL.Path, wildcard) {
			h.handler.ServeHTTP(w, req)
			return
		}
	}

	// If the token is valid, allow the request
	// TOOD: UPDATE TO VERIFY AGAINST RESOURCE
	// if _, err := h.auth.Verify(token); err == nil {
	// 	h.handler.ServeHTTP(w, req)
	// 	return
	// }

	// If there is no auth login url set, 401
	if loginURL == "" {
		w.WriteHeader(401)
		return
	}

	// Redirect to the login path
	params := url.Values{"redirect_to": {req.URL.Path}}
	loginWithRedirect := fmt.Sprintf("%v?%v", loginURL, params.Encode())
	http.Redirect(w, req, loginWithRedirect, http.StatusTemporaryRedirect)
}
