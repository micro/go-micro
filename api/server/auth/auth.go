package auth

import (
	"net/http"
	"strings"

	"github.com/micro/go-micro/v2/auth"
	"github.com/micro/go-micro/v2/metadata"
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
	loginURL := h.auth.Options().LoginURL

	// Return if the user disabled auth on this endpoint
	excludes := h.auth.Options().Exclude
	if len(loginURL) > 0 {
		excludes = append(excludes, loginURL)
	}
	for _, e := range excludes {
		if e == req.URL.Path {
			h.handler.ServeHTTP(w, req)
			return
		}
	}

	// Extract the auth token from the request
	var token string
	if header, ok := metadata.Get(req.Context(), "Authorization"); ok {
		if strings.HasPrefix(header, BearerScheme) {
			token = header[len(BearerScheme):]
		}
	}

	// If the token is valid, allow the request
	if _, err := h.auth.Verify(token); err == nil {
		h.handler.ServeHTTP(w, req)
		return
	}

	// If there is no auth login url set, 401
	if loginURL == "" {
		w.WriteHeader(401)
	}

	// Redirect to the login path
	http.Redirect(w, req, loginURL, http.StatusTemporaryRedirect)
}
