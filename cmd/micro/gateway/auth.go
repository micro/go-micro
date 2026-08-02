package gateway

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
)

// authToken is the static machine token accepted as an admin ("*" scope)
// credential, alongside JWTs. Empty means JWT-only. It is minted (or supplied)
// once per process — never a fixed default like admin/micro.
var authToken string

// isExposed reports whether a bind address is reachable beyond loopback. An
// empty host (":8080"), 0.0.0.0, and :: bind all interfaces and are exposed;
// 127.0.0.1 / localhost / ::1 are not. Unclassifiable hostnames are treated as
// exposed (fail safe).
func isExposed(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = strings.Trim(addr, "[]")
	}
	switch strings.ToLower(host) {
	case "", "0.0.0.0", "::":
		return true
	case "localhost":
		return false
	}
	if ip := net.ParseIP(host); ip != nil {
		return !ip.IsLoopback()
	}
	return true
}

// AuthFlags are the auth-policy flags shared by `micro run` and `micro gateway`.
func AuthFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:  "auth",
			Usage: "Force authentication on (default: on when the bind address is non-loopback)",
		},
		&cli.BoolFlag{
			Name:  "no-auth",
			Usage: "Force authentication off",
		},
		&cli.StringFlag{
			Name:    "auth-token",
			Usage:   "Static bearer token to require; if omitted while auth is on, one is generated and printed once",
			EnvVars: []string{"MICRO_AUTH_TOKEN"},
		},
	}
}

// ResolveAuth decides whether the gateway on addr requires authentication and
// provisions the machine token. The default follows the socket — auth is on
// when addr is exposed, off on loopback — and is overridable with --auth /
// --no-auth or MICRO_AUTH=on|off. A token is always provisioned (supplied via
// --auth-token/MICRO_AUTH_TOKEN, else generated) so that scoped/paid tools stay
// reachable even in auth-off mode; the returned string is non-empty only when a
// token was generated and should be printed once.
func ResolveAuth(c *cli.Context, addr string) (enabled bool, generatedToken string) {
	switch strings.ToLower(authOverride(c)) {
	case "off", "false", "no", "0":
		enabled = false
	case "on", "true", "yes", "1":
		enabled = true
	default:
		enabled = isExposed(addr)
	}

	if tok := c.String("auth-token"); tok != "" {
		authToken = tok
	} else if authToken == "" {
		authToken = generateToken()
		generatedToken = authToken
	}
	return enabled, generatedToken
}

func authOverride(c *cli.Context) string {
	if c.Bool("no-auth") {
		return "off"
	}
	if c.Bool("auth") {
		return "on"
	}
	return os.Getenv("MICRO_AUTH")
}

func generateToken() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand should never fail; fall back is still unique per process.
		return hex.EncodeToString([]byte(os.Args[0]))
	}
	return hex.EncodeToString(b)
}

// tokenMatches reports whether tok equals the static machine token in constant
// time. An empty static token or empty tok never matches.
func tokenMatches(tok string) bool {
	if authToken == "" || tok == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(tok), []byte(authToken)) == 1
}

// extractToken pulls a bearer token from the request: Authorization header
// first, then a `token` query parameter (for SSE / browser links), then the
// micro_token cookie.
func extractToken(r *http.Request) string {
	if authz := r.Header.Get("Authorization"); strings.HasPrefix(authz, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(authz, "Bearer "))
	}
	if q := r.URL.Query().Get("token"); q != "" {
		return q
	}
	if cookie, err := r.Cookie("micro_token"); err == nil {
		return cookie.Value
	}
	return ""
}
