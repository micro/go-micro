package cors

import (
	"net/http"
)

type Config struct {
	AllowOrigin      string
	AllowCredentials bool
	AllowMethods     string
	AllowHeaders     string
}

// CombinedCORSHandler wraps a server and provides CORS headers.
func CombinedCORSHandler(h http.Handler, config *Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if config != nil {
			SetHeaders(w, r, config)
		}
		if r.Method == "OPTIONS" {
			return
		}

		h.ServeHTTP(w, r)
	})
}

// SetHeaders sets the CORS headers.
func SetHeaders(w http.ResponseWriter, _ *http.Request, config *Config) {
	set := func(w http.ResponseWriter, k, v string) {
		if v := w.Header().Get(k); len(v) > 0 {
			return
		}
		w.Header().Set(k, v)
	}
	// For forward-compatible code, default values may not be provided in the future
	if config.AllowCredentials {
		set(w, "Access-Control-Allow-Credentials", "true")
	} else {
		set(w, "Access-Control-Allow-Credentials", "false")
	}
	if config.AllowOrigin == "" {
		set(w, "Access-Control-Allow-Origin", "*")
	} else {
		set(w, "Access-Control-Allow-Origin", config.AllowOrigin)
	}
	if config.AllowMethods == "" {
		set(w, "Access-Control-Allow-Methods", "POST, PATCH, GET, OPTIONS, PUT, DELETE")
	} else {
		set(w, "Access-Control-Allow-Methods", config.AllowMethods)
	}
	if config.AllowHeaders == "" {
		set(w, "Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	} else {
		set(w, "Access-Control-Allow-Headers", config.AllowHeaders)
	}
}
