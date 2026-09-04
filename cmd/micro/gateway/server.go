package gateway

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"text/template"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/cli/v2"
	"go-micro.dev/v6/ai"
	_ "go-micro.dev/v6/ai/anthropic"
	_ "go-micro.dev/v6/ai/atlascloud"
	_ "go-micro.dev/v6/ai/gemini"
	_ "go-micro.dev/v6/ai/groq"
	_ "go-micro.dev/v6/ai/mistral"
	_ "go-micro.dev/v6/ai/openai"
	_ "go-micro.dev/v6/ai/together"
	"go-micro.dev/v6/auth"
	"go-micro.dev/v6/auth/jwt"
	"go-micro.dev/v6/client"
	"go-micro.dev/v6/cmd"
	codecBytes "go-micro.dev/v6/codec/bytes"
	"go-micro.dev/v6/gateway/mcp"
	"go-micro.dev/v6/metadata"
	"go-micro.dev/v6/registry"
	"go-micro.dev/v6/store"
	"go-micro.dev/v6/wrapper/x402"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sync/errgroup"
)

// HTML is the embedded filesystem for templates and static files, set by main.go
var HTML fs.FS

const agentSystemPrompt = "You are an agent that helps users interact with microservices. Use the available tools to fulfill user requests. When you call a tool, explain what you are doing."

var (
	apiCache struct {
		sync.Mutex
		data map[string]any
		time time.Time
	}
)

type templates struct {
	api        *template.Template
	service    *template.Template
	form       *template.Template
	home       *template.Template
	logs       *template.Template
	log        *template.Template
	status     *template.Template
	authTokens *template.Template
	authLogin  *template.Template
	authUsers  *template.Template
	playground *template.Template
	scopes     *template.Template
}
type TemplateUser struct {
	ID string
}

// Account is an alias for auth.Account from the framework.
// The gateway stores accounts in the default store under "auth/<id>" keys.
// Scopes on accounts are checked against endpoint-scopes by checkEndpointScopes.
type Account = auth.Account

func parseTemplates() *templates {
	return &templates{
		api:        template.Must(template.ParseFS(HTML, "web/templates/base.html", "web/templates/api.html")),
		service:    template.Must(template.ParseFS(HTML, "web/templates/base.html", "web/templates/service.html")),
		form:       template.Must(template.ParseFS(HTML, "web/templates/base.html", "web/templates/form.html")),
		home:       template.Must(template.ParseFS(HTML, "web/templates/base.html", "web/templates/home.html")),
		logs:       template.Must(template.ParseFS(HTML, "web/templates/base.html", "web/templates/logs.html")),
		log:        template.Must(template.ParseFS(HTML, "web/templates/base.html", "web/templates/log.html")),
		status:     template.Must(template.ParseFS(HTML, "web/templates/base.html", "web/templates/status.html")),
		authTokens: template.Must(template.ParseFS(HTML, "web/templates/base.html", "web/templates/auth_tokens.html")),
		authLogin:  template.Must(template.ParseFS(HTML, "web/templates/base.html", "web/templates/auth_login.html")),
		authUsers:  template.Must(template.ParseFS(HTML, "web/templates/base.html", "web/templates/auth_users.html")),
		playground: template.Must(template.ParseFS(HTML, "web/templates/base.html", "web/templates/playground.html")),
		scopes:     template.Must(template.ParseFS(HTML, "web/templates/base.html", "web/templates/scopes.html")),
	}
}

// Helper to extract user info from JWT cookie
func getUser(r *http.Request) string {
	cookie, err := r.Cookie("micro_token")
	if err != nil || cookie.Value == "" {
		return ""
	}
	// Parse JWT claims (just decode, don't verify)
	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 3 {
		return ""
	}
	payload, err := decodeSegment(parts[1])
	if err != nil {
		return ""
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}
	if sub, ok := claims["sub"].(string); ok {
		return sub
	}
	if id, ok := claims["id"].(string); ok {
		return id
	}
	return ""
}

// Helper to decode JWT base64url segment
func decodeSegment(seg string) ([]byte, error) {
	// JWT uses base64url, no padding
	missing := len(seg) % 4
	if missing != 0 {
		seg += strings.Repeat("=", 4-missing)
	}
	return decodeBase64Url(seg)
}

func decodeBase64Url(s string) ([]byte, error) {
	return base64.URLEncoding.DecodeString(s)
}

// Helper: store JWT token
func storeJWTToken(storeInst store.Store, token, userID string) {
	_ = storeInst.Write(&store.Record{Key: "jwt/" + token, Value: []byte(userID)})
}

// Helper: check if JWT token is revoked (not present in store)
func isTokenRevoked(storeInst store.Store, token string) bool {
	recs, _ := storeInst.Read("jwt/" + token)
	return len(recs) == 0
}

// Helper: delete all JWT tokens for a user
func deleteUserTokens(storeInst store.Store, userID string) {
	recs, _ := storeInst.Read("jwt/", store.ReadPrefix())
	for _, rec := range recs {
		if string(rec.Value) == userID {
			_ = storeInst.Delete(rec.Key)
		}
	}
}

// Updated authRequired to accept storeInst as argument
func authRequired(storeInst store.Store) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token == "" {
				if strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/api" && r.URL.Path != "/api/" {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte(`{"error":"missing or invalid token"}`))
					return
				}
				// For API endpoints, return 401. For UI, redirect to login.
				if strings.HasPrefix(r.URL.Path, "/api/") {
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte("Unauthorized: missing token"))
					return
				}
				http.Redirect(w, r, "/auth/login", http.StatusFound)
				return
			}
			// A matching static machine token authenticates as admin.
			if tokenMatches(token) {
				next(w, r)
				return
			}
			claims, err := ParseJWT(token)
			if err != nil {
				if strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/api" && r.URL.Path != "/api/" {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte(`{"error":"invalid token"}`))
					return
				}
				if strings.HasPrefix(r.URL.Path, "/api/") {
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte("Unauthorized: invalid token"))
					return
				}
				http.Redirect(w, r, "/auth/login", http.StatusFound)
				return
			}
			if exp, ok := claims["exp"].(float64); ok {
				if int64(exp) < time.Now().Unix() {
					if strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/api" && r.URL.Path != "/api/" {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusUnauthorized)
						w.Write([]byte(`{"error":"token expired"}`))
						return
					}
					if strings.HasPrefix(r.URL.Path, "/api/") {
						w.WriteHeader(http.StatusUnauthorized)
						w.Write([]byte("Unauthorized: token expired"))
						return
					}
					http.Redirect(w, r, "/auth/login", http.StatusFound)
					return
				}
			}
			// Check for token revocation
			if isTokenRevoked(storeInst, token) {
				if strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/api" && r.URL.Path != "/api/" {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte(`{"error":"token revoked"}`))
					return
				}
				if strings.HasPrefix(r.URL.Path, "/api/") {
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte("Unauthorized: token revoked"))
					return
				}
				http.Redirect(w, r, "/auth/login", http.StatusFound)
				return
			}
			next(w, r)
		}
	}
}

func wrapAuth(authRequired func(http.HandlerFunc) http.HandlerFunc) func(http.HandlerFunc) http.HandlerFunc {
	return func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if strings.HasPrefix(path, "/auth/login") || strings.HasPrefix(path, "/auth/logout") ||
				path == "/styles.css" || path == "/main.js" {
				h(w, r)
				return
			}
			authRequired(h)(w, r)
		}
	}
}

// registryServices returns the set of currently-registered service names.
func registryServices() map[string]bool {
	set := map[string]bool{}
	if svcs, err := registry.ListServices(); err == nil {
		for _, s := range svcs {
			set[s.Name] = true
		}
	}
	return set
}

func getDashboardData() (serviceCount, runningCount, stoppedCount int, statusDot string) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}
	pidDir := homeDir + "/micro/run"
	dirEntries, err := os.ReadDir(pidDir)
	if err != nil {
		return
	}
	// A service is "running" if it heartbeats in the registry. The PID signal
	// check can't work across containers, so never rely on local processes.
	running := registryServices()
	for _, entry := range dirEntries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".pid") || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		pidFile := pidDir + "/" + entry.Name()
		pidBytes, err := os.ReadFile(pidFile)
		if err != nil {
			continue
		}
		lines := strings.Split(string(pidBytes), "\n")
		name := ""
		if len(lines) > 2 {
			name = lines[2]
		}
		serviceCount++
		if name != "" && running[name] {
			runningCount++
		} else {
			stoppedCount++
		}
	}
	if serviceCount > 0 && runningCount == serviceCount {
		statusDot = "green"
	} else if serviceCount > 0 && runningCount > 0 {
		statusDot = "yellow"
	} else {
		statusDot = "red"
	}
	return
}

// defaultAgentSettings returns the agent chat prefill from the gateway's own
// AI environment variables, used until the user saves settings via POST.
func defaultAgentSettings() map[string]string {
	return map[string]string{
		"provider": os.Getenv("MICRO_AI_PROVIDER"),
		"model":    os.Getenv("MICRO_AI_MODEL"),
		"api_key":  os.Getenv("MICRO_AI_API_KEY"),
		"base_url": os.Getenv("MICRO_AI_BASE_URL"),
	}
}

func registerHandlers(mux *http.ServeMux, tmpls *templates, storeInst store.Store, authEnabled bool) {
	var wrap func(http.HandlerFunc) http.HandlerFunc

	if authEnabled {
		authMw := authRequired(storeInst)
		wrap = wrapAuth(authMw)
	} else {
		// No auth in dev mode - pass through handlers unchanged
		wrap = func(h http.HandlerFunc) http.HandlerFunc {
			return h
		}
	}

	// renderPage injects AuthEnabled into template data so the sidebar can
	// conditionally show/hide auth links.
	renderPage := func(w http.ResponseWriter, tmpl *template.Template, data map[string]any) error {
		data["AuthEnabled"] = authEnabled
		return tmpl.Execute(w, data)
	}

	// checkEndpointScopes verifies the caller's token scopes against the
	// required scopes for a service endpoint. Returns true if allowed.
	// If not allowed, writes a 403 response and returns false.
	checkEndpointScopes := func(w http.ResponseWriter, r *http.Request, endpointKey string) bool {
		// Scope enforcement is independent of the auth on/off decision: an
		// endpoint that declares a required scope always needs a token bearing
		// it, even on a loopback gateway with auth off. That is what keeps
		// action/paid tools gated locally while read-only tools stay open.
		recs, _ := storeInst.Read("endpoint-scopes/" + endpointKey)
		if len(recs) == 0 {
			return true // no scopes configured = unrestricted
		}
		var requiredScopes []string
		if err := json.Unmarshal(recs[0].Value, &requiredScopes); err != nil || len(requiredScopes) == 0 {
			return true
		}
		// A matching static machine token is admin (all scopes).
		token := extractToken(r)
		if tokenMatches(token) {
			return true
		}
		// Extract caller's scopes from JWT
		callerScopes := []string{}
		if token != "" {
			if claims, err := ParseJWT(token); err == nil {
				if s, ok := claims["scopes"].([]interface{}); ok {
					for _, v := range s {
						if str, ok := v.(string); ok {
							callerScopes = append(callerScopes, str)
						}
					}
				}
			}
		}
		for _, cs := range callerScopes {
			if cs == "*" {
				return true
			}
			for _, rs := range requiredScopes {
				if cs == rs {
					return true
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"error":           "insufficient scopes",
			"required_scopes": strings.Join(requiredScopes, ","),
		})
		return false
	}

	// Prometheus scrape endpoint (Alloy scrapes micro-run:8080)
	mux.Handle("/metrics", promhttp.Handler())

	// pprof endpoints on the gateway mux so Pyroscope (via Alloy) can scrape
	// continuous profiles from the compose gateway without a dedicated :6060
	// listener (that only exists under `micro run`, not in docker).
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	// Serve static files with correct Content-Type
	mux.HandleFunc("/styles.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		f, err := HTML.Open("web/styles.css")
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		defer f.Close()
		_, _ = io.Copy(w, f)
	})

	mux.HandleFunc("/main.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		f, err := HTML.Open("web/main.js")
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		defer f.Close()
		_, _ = io.Copy(w, f)
	})

	// MCP API endpoints - list tools and call tools through the web server
	mux.HandleFunc("/mcp/tools", wrap(func(w http.ResponseWriter, r *http.Request) {
		services, err := registry.ListServices()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		var tools []map[string]any
		for _, svc := range services {
			fullSvcs, err := registry.GetService(svc.Name)
			if err != nil || len(fullSvcs) == 0 {
				continue
			}
			for _, ep := range fullSvcs[0].Endpoints {
				toolName := fmt.Sprintf("%s.%s", svc.Name, ep.Name)
				description := fmt.Sprintf("Call %s on %s service", ep.Name, svc.Name)
				if ep.Metadata != nil {
					if desc, ok := ep.Metadata["description"]; ok && desc != "" {
						description = desc
					}
				}
				inputSchema := map[string]any{
					"type":       "object",
					"properties": map[string]any{},
				}
				if ep.Request != nil && len(ep.Request.Values) > 0 {
					props := inputSchema["properties"].(map[string]any)
					for _, field := range ep.Request.Values {
						props[field.Name] = map[string]any{
							"type":        mapGoTypeToJSON(field.Type),
							"description": fmt.Sprintf("%s field", field.Name),
						}
					}
				}
				tool := map[string]any{
					"name":        toolName,
					"description": description,
					"inputSchema": inputSchema,
				}
				// Extract scopes from endpoint metadata or store
				if ep.Metadata != nil {
					if scopes, ok := ep.Metadata["scopes"]; ok && scopes != "" {
						tool["scopes"] = strings.Split(scopes, ",")
					}
				}
				// Override with stored scopes (from UI) if present
				if recs, _ := storeInst.Read("endpoint-scopes/" + toolName); len(recs) > 0 {
					var storedScopes []string
					if err := json.Unmarshal(recs[0].Value, &storedScopes); err == nil && len(storedScopes) > 0 {
						tool["scopes"] = storedScopes
					}
				}
				tools = append(tools, tool)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"tools": tools})
	}))

	mux.HandleFunc("/mcp/call", wrap(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
			return
		}
		var req struct {
			Tool  string         `json:"tool"`
			Input map[string]any `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		// Parse tool name into service and endpoint
		parts := strings.SplitN(req.Tool, ".", 2)
		if len(parts) != 2 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid tool name, expected service.endpoint"})
			return
		}
		serviceName := parts[0]
		endpointName := parts[1]

		// Check endpoint scopes
		if !checkEndpointScopes(w, r, req.Tool) {
			return
		}

		// Build RPC request using default client
		inputBytes, err := json.Marshal(req.Input)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		rpcReq := client.DefaultClient.NewRequest(serviceName, endpointName, &codecBytes.Frame{Data: inputBytes})
		var rsp codecBytes.Frame
		if err := client.DefaultClient.Call(r.Context(), rpcReq, &rsp); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("RPC call failed: %v", err)})
			return
		}

		var traceBytes [16]byte
		_, _ = rand.Read(traceBytes[:])
		traceID := fmt.Sprintf("%x", traceBytes)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"result":   json.RawMessage(rsp.Data),
			"trace_id": traceID,
		})
	}))

	// Agent settings endpoints
	mux.HandleFunc("/api/agent/settings", wrap(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			recs, _ := storeInst.Read("agent/settings")
			if len(recs) == 0 {
				json.NewEncoder(w).Encode(defaultAgentSettings())
				return
			}
			var settings map[string]string
			if err := json.Unmarshal(recs[0].Value, &settings); err != nil {
				log.Printf("[agent] failed to parse settings: %v", err)
				json.NewEncoder(w).Encode(defaultAgentSettings())
				return
			}
			json.NewEncoder(w).Encode(settings)
			return
		}
		if r.Method == http.MethodPost {
			var settings map[string]string
			if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			b, _ := json.Marshal(settings)
			_ = storeInst.Write(&store.Record{Key: "agent/settings", Value: b})
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
	}))

	// Agent prompt endpoint — sends user prompt to LLM with tool definitions
	mux.HandleFunc("/api/agent/prompt", wrap(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
			return
		}
		var req struct {
			Prompt string `json:"prompt"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		// Load settings
		recs, _ := storeInst.Read("agent/settings")
		var settings map[string]string
		if len(recs) > 0 {
			if err := json.Unmarshal(recs[0].Value, &settings); err != nil {
				log.Printf("[agent] failed to parse settings: %v", err)
			}
		}
		apiKey := ""
		modelName := ""
		baseURL := ""
		provider := ""
		if settings != nil {
			if v := settings["api_key"]; v != "" {
				apiKey = v
			}
			if v := settings["model"]; v != "" {
				modelName = v
			}
			if v := settings["base_url"]; v != "" {
				baseURL = v
			}
			if v := settings["provider"]; v != "" {
				provider = v
			}
		}
		if apiKey == "" {
			apiKey = os.Getenv("MICRO_AI_API_KEY")
		}
		if modelName == "" {
			modelName = os.Getenv("MICRO_AI_MODEL")
		}
		if baseURL == "" {
			baseURL = os.Getenv("MICRO_AI_BASE_URL")
		}
		if provider == "" {
			provider = os.Getenv("MICRO_AI_PROVIDER")
		}
		if provider == "" && baseURL == "" {
			json.NewEncoder(w).Encode(map[string]string{"error": "No provider configured. Go to Agent settings to add one."})
			return
		}

		// Auto-detect provider if not explicitly set
		if provider == "" {
			provider = ai.AutoDetectProvider(baseURL)
		}

		// Discover tools from registry
		services, _ := registry.ListServices()
		var discoveredTools []ai.Tool
		// safeNameMap maps LLM-safe names back to original dotted names
		safeNameMap := map[string]string{}
		for _, svc := range services {
			fullSvcs, err := registry.GetService(svc.Name)
			if err != nil || len(fullSvcs) == 0 {
				continue
			}
			for _, ep := range fullSvcs[0].Endpoints {
				tName := fmt.Sprintf("%s.%s", svc.Name, ep.Name)
				safeName := strings.ReplaceAll(tName, ".", "_")
				safeNameMap[safeName] = tName
				desc := fmt.Sprintf("Call %s on %s service", ep.Name, svc.Name)
				if ep.Metadata != nil {
					if d, ok := ep.Metadata["description"]; ok && d != "" {
						desc = d
					}
				}
				props := map[string]any{}
				if ep.Request != nil {
					for _, field := range ep.Request.Values {
						props[field.Name] = map[string]any{
							"type":        mapGoTypeToJSON(field.Type),
							"description": fmt.Sprintf("%s (%s)", field.Name, field.Type),
						}
					}
				}
				discoveredTools = append(discoveredTools, ai.Tool{
					Name:         safeName,
					OriginalName: tName,
					Description:  desc,
					Properties:   props,
				})
			}
		}

		// executeToolCall runs an RPC tool call and returns the result.
		// toolName can be either the original dotted name or the LLM-safe
		// underscored name; the safe name is resolved first.
		// Checks endpoint scopes against the caller's token before executing.
		executeToolCall := func(_ context.Context, call ai.ToolCall) ai.ToolResult {
			toolName := call.Name
			input := call.Input
			if orig, ok := safeNameMap[toolName]; ok {
				toolName = orig
			}
			// Check endpoint scopes
			if authEnabled {
				recs, _ := storeInst.Read("endpoint-scopes/" + toolName)
				if len(recs) > 0 {
					var requiredScopes []string
					if err := json.Unmarshal(recs[0].Value, &requiredScopes); err == nil && len(requiredScopes) > 0 {
						// Get caller's scopes from JWT
						callerScopes := []string{}
						token := ""
						if authz := r.Header.Get("Authorization"); strings.HasPrefix(authz, "Bearer ") {
							token = strings.TrimPrefix(authz, "Bearer ")
						}
						if token == "" {
							if cookie, err := r.Cookie("micro_token"); err == nil {
								token = cookie.Value
							}
						}
						if token != "" {
							if claims, err := ParseJWT(token); err == nil {
								if s, ok := claims["scopes"].([]interface{}); ok {
									for _, v := range s {
										if str, ok := v.(string); ok {
											callerScopes = append(callerScopes, str)
										}
									}
								}
							}
						}
						allowed := false
						for _, cs := range callerScopes {
							if cs == "*" {
								allowed = true
								break
							}
							for _, rs := range requiredScopes {
								if cs == rs {
									allowed = true
									break
								}
							}
							if allowed {
								break
							}
						}
						if !allowed {
							errMsg := fmt.Sprintf(`{"error":"insufficient scopes","required_scopes":"%s"}`, strings.Join(requiredScopes, ","))
							return ai.ToolResult{ID: call.ID, Value: map[string]string{"error": "insufficient scopes", "required_scopes": strings.Join(requiredScopes, ",")}, Content: errMsg}
						}
					}
				}
			}
			parts := strings.SplitN(toolName, ".", 2)
			if len(parts) != 2 {
				errMsg := `{"error":"invalid tool name"}`
				return ai.ToolResult{ID: call.ID, Value: map[string]string{"error": "invalid tool name"}, Content: errMsg}
			}
			inputBytes, _ := json.Marshal(input)
			rpcReq := client.DefaultClient.NewRequest(parts[0], parts[1], &codecBytes.Frame{Data: inputBytes})
			var rsp codecBytes.Frame
			if err := client.DefaultClient.Call(r.Context(), rpcReq, &rsp); err != nil {
				errMsg := fmt.Sprintf(`{"error":"%s"}`, err.Error())
				return ai.ToolResult{ID: call.ID, Value: map[string]string{"error": err.Error()}, Content: errMsg}
			}
			var rpcResult any
			if err := json.Unmarshal(rsp.Data, &rpcResult); err != nil {
				rpcResult = string(rsp.Data)
			}
			return ai.ToolResult{ID: call.ID, Value: rpcResult, Content: string(rsp.Data)}
		}

		// Create model with options
		var modelOpts []ai.Option
		modelOpts = append(modelOpts, ai.WithAPIKey(apiKey))
		if modelName != "" {
			modelOpts = append(modelOpts, ai.WithModel(modelName))
		}
		if baseURL != "" {
			modelOpts = append(modelOpts, ai.WithBaseURL(baseURL))
		}
		modelOpts = append(modelOpts, ai.WithToolHandler(executeToolCall))

		m := ai.New(provider, modelOpts...)
		if m == nil {
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create model provider"})
			return
		}

		// Build request
		modelReq := &ai.Request{
			Prompt:       req.Prompt,
			SystemPrompt: agentSystemPrompt,
			Tools:        discoveredTools,
		}

		// Generate response
		response, err := m.Generate(r.Context(), modelReq)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		// Build result
		result := map[string]any{}
		if response.Reply != "" {
			result["reply"] = response.Reply
		}
		if len(response.ToolCalls) > 0 {
			var toolCalls []map[string]any
			for _, tc := range response.ToolCalls {
				var result any
				if tc.Result != "" {
					if err := json.Unmarshal([]byte(tc.Result), &result); err != nil {
						result = tc.Result
					}
				}
				toolCalls = append(toolCalls, map[string]any{
					"tool":   tc.Name,
					"input":  tc.Input,
					"result": result,
					"error":  tc.Error,
				})
			}
			result["tool_calls"] = toolCalls
		}
		if response.Answer != "" {
			result["answer"] = response.Answer
		}

		json.NewEncoder(w).Encode(result)
	}))

	mux.HandleFunc("/", wrap(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasPrefix(path, "/auth/") {
			// Let the dedicated /auth/* handlers process this
			return
		}
		userID := getUser(r)
		var user any
		if userID != "" {
			user = &TemplateUser{ID: userID}
		} else {
			user = nil
		}
		if path == "/" {
			serviceCount, runningCount, stoppedCount, statusDot := getDashboardData()
			// Fetch registered services for the home page
			var homeServices []string
			if svcs, err := registry.ListServices(); err == nil {
				for _, s := range svcs {
					homeServices = append(homeServices, s.Name)
				}
				sort.Strings(homeServices)
			}
			err := renderPage(w, tmpls.home, map[string]any{
				"Title":        "Home",
				"WebLink":      "/",
				"ServiceCount": serviceCount,
				"RunningCount": runningCount,
				"StoppedCount": stoppedCount,
				"StatusDot":    statusDot,
				"Services":     homeServices,
				"User":         user,
			})
			if err != nil {
				log.Printf("[TEMPLATE ERROR] home: %v", err)
			}
			return
		}
		if path == "/api" || path == "/api/" {
			apiCache.Lock()
			useCache := false
			if apiCache.data != nil && time.Since(apiCache.time) < 30*time.Second {
				useCache = true
			}
			var apiData map[string]any
			var sidebarEndpoints []map[string]string
			if useCache {
				apiData = apiCache.data
				if v, ok := apiData["SidebarEndpoints"]; ok {
					sidebarEndpoints, _ = v.([]map[string]string)
				}
			} else {
				services, _ := registry.ListServices()
				var apiServices []map[string]any
				for _, srv := range services {
					srvs, err := registry.GetService(srv.Name)
					if err != nil || len(srvs) == 0 {
						continue
					}
					s := srvs[0]
					if len(s.Endpoints) == 0 {
						continue
					}
					endpoints := []map[string]any{}
					for _, ep := range s.Endpoints {
						parts := strings.Split(ep.Name, ".")
						if len(parts) != 2 {
							continue
						}
						apiPath := fmt.Sprintf("/%s/%s", s.Name, ep.Name)
						var params, response string
						if ep.Request != nil && len(ep.Request.Values) > 0 {
							params += "<ul class=no-bullets>"
							for _, v := range ep.Request.Values {
								params += fmt.Sprintf("<li><b>%s</b> <span style='color:#888;'>%s</span></li>", v.Name, v.Type)
							}
							params += "</ul>"
						} else {
							params = "<i style='color:#888;'>No parameters</i>"
						}
						if ep.Response != nil && len(ep.Response.Values) > 0 {
							response += "<ul class=no-bullets>"
							for _, v := range ep.Response.Values {
								response += fmt.Sprintf("<li><b>%s</b> <span style='color:#888;'>%s</span></li>", v.Name, v.Type)
							}
							response += "</ul>"
						} else {
							response = "<i style='color:#888;'>No response fields</i>"
						}
						endpoints = append(endpoints, map[string]any{
							"Name":     ep.Name,
							"Path":     apiPath,
							"Params":   params,
							"Response": response,
						})
					}
					anchor := strings.ReplaceAll(s.Name, ".", "-")
					apiServices = append(apiServices, map[string]any{
						"Name":      s.Name,
						"Anchor":    anchor,
						"Endpoints": endpoints,
					})
					sidebarEndpoints = append(sidebarEndpoints, map[string]string{"Name": s.Name, "Anchor": anchor})
				}
				sort.Slice(sidebarEndpoints, func(i, j int) bool {
					return sidebarEndpoints[i]["Name"] < sidebarEndpoints[j]["Name"]
				})
				apiData = map[string]any{"Title": "API", "WebLink": "/", "Services": apiServices, "SidebarEndpoints": sidebarEndpoints, "SidebarEndpointsEnabled": true, "User": user}

				apiCache.data = apiData
				apiCache.time = time.Now()
			}
			apiCache.Unlock()
			// Add API auth doc at the top — reflects the address-based policy.
			if authEnabled {
				apiData["ApiAuthDoc"] = `<div style='background:#f8f8e8; border:1px solid #e0e0b0; padding:1em; margin-bottom:2em; font-size:1.08em;'>
<b>Authentication required:</b> this gateway is exposed, so all <code>/api/...</code> calls must include an <b>Authorization: Bearer &lt;token&gt;</b> header (or <code>?token=</code>). <br>
Use the token printed at startup, or generate more on the <a href='/auth/tokens'>Tokens page</a>.
</div>`
			} else {
				apiData["ApiAuthDoc"] = `<div style='background:#eef6ee; border:1px solid #bcd8bc; padding:1em; margin-bottom:2em; font-size:1.08em;'>
<b>Auth is off</b> (gateway on loopback) — call <code>/api/...</code> directly, no token needed. Tools with a required <a href='/auth/scopes'>scope</a> (actions, paid) still need a token.
</div>`
			}
			_ = renderPage(w, tmpls.api, apiData)
			return
		}
		// HTTP->RPC proxy: /api/{service}/{method} (e.g. /api/helloworld/Helloworld.Call)
		// also accepts /api/{service}/{pkg}/{method} as linked from the API explorer.
		if strings.HasPrefix(path, "/api/") {
			parts := strings.Split(strings.TrimPrefix(path, "/api/"), "/")
			var service, endpoint string
			switch len(parts) {
			case 2:
				service, endpoint = parts[0], parts[1]
			case 3:
				service, endpoint = parts[0], parts[1]+"."+parts[2]
			default:
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte("Not found"))
				return
			}
			// Trace the proxy call so gateway activity shows up in Tempo
			// alongside the agent/flow spans it fans out to. A caller's
			// traceparent header continues the trace.
			ctx, span := otel.Tracer("go-micro.dev/v6/gateway/api").Start(
				otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header)),
				"api.proxy",
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					attribute.String("rpc.service", service),
					attribute.String("rpc.method", endpoint),
				),
			)
			spanErr := error(nil)
			defer func() {
				if spanErr != nil {
					span.RecordError(spanErr)
					span.SetStatus(codes.Error, spanErr.Error())
				} else {
					span.SetStatus(codes.Ok, "")
				}
				span.End()
			}()
			svcs, err := registry.GetService(service)
			if err != nil || len(svcs) == 0 {
				spanErr = fmt.Errorf("service %q not found", service)
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte("Service not found: " + service))
				return
			}
			valid := false
			for _, ep := range svcs[0].Endpoints {
				if ep.Name == endpoint {
					valid = true
					break
				}
			}
			if !valid {
				spanErr = fmt.Errorf("endpoint %q not found", endpoint)
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte("Endpoint not found: " + endpoint))
				return
			}
			if !checkEndpointScopes(w, r, service+"."+endpoint) {
				spanErr = fmt.Errorf("endpoint %s blocked by scopes", service+"."+endpoint)
				return
			}
			inputBytes, err := io.ReadAll(r.Body)
			if err != nil {
				spanErr = err
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(err.Error()))
				return
			}
			// Carry the proxy span's trace context in the RPC metadata so an
			// instrumented target service's spans nest under this one.
			md, _ := metadata.FromContext(ctx)
			if md == nil {
				md = metadata.Metadata{}
			}
			carrier := make(propagation.MapCarrier)
			otel.GetTextMapPropagator().Inject(ctx, carrier)
			for k, v := range carrier {
				md.Set(strings.Title(k), v)
			}
			ctx = metadata.NewContext(ctx, md)
			rpcReq := client.DefaultClient.NewRequest(service, endpoint, &codecBytes.Frame{Data: inputBytes})
			var rsp codecBytes.Frame
			if err := client.DefaultClient.Call(ctx, rpcReq, &rsp); err != nil {
				spanErr = err
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadGateway)
				w.Write([]byte(`{"error":` + strconv.Quote(err.Error()) + `}`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(rsp.Data)
			return
		}
		if path == "/services" {
			// Do NOT include SidebarEndpoints on this page
			services, _ := registry.ListServices()
			var serviceNames []string
			for _, service := range services {
				serviceNames = append(serviceNames, service.Name)
			}
			sort.Strings(serviceNames)
			_ = renderPage(w, tmpls.service, map[string]any{"Title": "Services", "WebLink": "/", "Services": serviceNames, "User": user})

			return
		}
		if path == "/agent" {
			_ = renderPage(w, tmpls.playground, map[string]any{"Title": "Agent", "WebLink": "/", "User": user})

			return
		}
		if path == "/logs" || path == "/logs/" {
			// Do NOT include SidebarEndpoints on this page
			homeDir, err := os.UserHomeDir()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Could not get home directory"))
				return
			}
			logsDir := homeDir + "/micro/logs"
			if err := os.MkdirAll(logsDir, 0755); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Could not create logs directory: " + err.Error()))
				return
			}
			dirEntries, err := os.ReadDir(logsDir)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Could not list logs directory: " + err.Error()))
				return
			}
			serviceNames := []string{}
			for _, entry := range dirEntries {
				name := entry.Name()
				if !entry.IsDir() && strings.HasSuffix(name, ".log") && !strings.HasPrefix(name, ".") {
					serviceNames = append(serviceNames, strings.TrimSuffix(name, ".log"))
				}
			}
			_ = renderPage(w, tmpls.logs, map[string]any{"Title": "Logs", "WebLink": "/", "Services": serviceNames, "User": user})
			return
		}
		if strings.HasPrefix(path, "/logs/") {
			// Do NOT include SidebarEndpoints on this page
			service := strings.TrimPrefix(path, "/logs/")
			if service == "" {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte("Service not specified"))
				return
			}
			homeDir, err := os.UserHomeDir()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Could not get home directory"))
				return
			}
			logFilePath := homeDir + "/micro/logs/" + service + ".log"
			f, err := os.Open(logFilePath)
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte("Could not open log file for service: " + service))
				return
			}
			defer f.Close()
			logBytes, err := io.ReadAll(f)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Could not read log file for service: " + service))
				return
			}
			logText := string(logBytes)
			_ = renderPage(w, tmpls.log, map[string]any{"Title": "Logs for " + service, "WebLink": "/logs", "Service": service, "Log": logText, "User": user})
			return
		}
		if path == "/status" {
			// Do NOT include SidebarEndpoints on this page
			homeDir, err := os.UserHomeDir()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Could not get home directory"))
				return
			}
			pidDir := homeDir + "/micro/run"
			if err := os.MkdirAll(pidDir, 0755); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Could not create pid directory: " + err.Error()))
				return
			}
			dirEntries, err := os.ReadDir(pidDir)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Could not list pid directory: " + err.Error()))
				return
			}
			statuses := []map[string]string{}
			running := registryServices()
			for _, entry := range dirEntries {
				if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".pid") || strings.HasPrefix(entry.Name(), ".") {
					continue
				}
				pidFile := pidDir + "/" + entry.Name()
				pidBytes, err := os.ReadFile(pidFile)
				if err != nil {
					statuses = append(statuses, map[string]string{
						"Service": entry.Name(),
						"Dir":     "-",
						"Status":  "unknown",
						"PID":     "-",
						"Uptime":  "-",
						"ID":      strings.TrimSuffix(entry.Name(), ".pid"),
					})
					continue
				}
				lines := strings.Split(string(pidBytes), "\n")
				pid := "-"
				dir := "-"
				service := "-"
				start := "-"
				if len(lines) > 0 && len(lines[0]) > 0 {
					pid = lines[0]
				}
				if len(lines) > 1 && len(lines[1]) > 0 {
					dir = lines[1]
				}
				if len(lines) > 2 && len(lines[2]) > 0 {
					service = lines[2]
				}
				if len(lines) > 3 && len(lines[3]) > 0 {
					start = lines[3]
				}
				// Running = heartbeats in the registry (pid signal checks cannot
				// reach processes in other containers).
				status := "stopped"
				if service != "" && running[service] {
					status = "running"
				}
				uptime := "-"
				if start != "-" {
					if t, err := parseStartTime(start); err == nil {
						uptime = time.Since(t).Truncate(time.Second).String()
					}
				}
				statuses = append(statuses, map[string]string{
					"Service": service,
					"Dir":     dir,
					"Status":  status,
					"PID":     pid,
					"Uptime":  uptime,
					"ID":      strings.TrimSuffix(entry.Name(), ".pid"),
				})
			}
			_ = renderPage(w, tmpls.status, map[string]any{"Title": "Status", "WebLink": "/", "Statuses": statuses, "User": user})
			return
		}
		// Match /{service} and /{service}/{endpoint}
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) >= 1 && parts[0] != "api" && parts[0] != "html" && parts[0] != "services" {
			service := parts[0]
			if len(parts) == 1 {
				s, err := registry.GetService(service)
				if err != nil || len(s) == 0 {
					w.WriteHeader(http.StatusNotFound)
					fmt.Fprintf(w, "Service not found: %s", service)
					return
				}
				endpoints := []map[string]string{}
				for _, ep := range s[0].Endpoints {
					endpoints = append(endpoints, map[string]string{
						"Name": ep.Name,
						"Path": fmt.Sprintf("/%s/%s", service, ep.Name),
					})
				}
				b, _ := json.MarshalIndent(s[0], "", "    ")
				_ = renderPage(w, tmpls.service, map[string]any{
					"Title":       "Service: " + service,
					"WebLink":     "/",
					"ServiceName": service,
					"Endpoints":   endpoints,
					"Description": string(b),
					"User":        user,
				})
				return
			}
			if len(parts) == 2 {
				service := parts[0]
				endpoint := parts[1] // Use the actual endpoint name from the URL, e.g. Foo.Bar
				s, err := registry.GetService(service)
				if err != nil || len(s) == 0 {
					w.WriteHeader(http.StatusNotFound)
					w.Write([]byte("Service not found: " + service))
					return
				}
				var ep *registry.Endpoint
				for _, eps := range s[0].Endpoints {
					if eps.Name == endpoint {
						ep = eps
						break
					}
				}
				if ep == nil {
					w.WriteHeader(http.StatusNotFound)
					w.Write([]byte("Endpoint not found"))
					return
				}
				if r.Method == http.MethodGet {
					// Build form fields from endpoint request values
					var inputs []map[string]string
					if ep.Request != nil && len(ep.Request.Values) > 0 {
						for _, input := range ep.Request.Values {
							inputs = append(inputs, map[string]string{
								"Label":       input.Name,
								"Name":        input.Name,
								"Placeholder": input.Name,
								"Value":       "",
							})
						}
					}
					_ = renderPage(w, tmpls.form, map[string]any{
						"Title":        "Service: " + service,
						"WebLink":      "/",
						"ServiceName":  service,
						"EndpointName": ep.Name,
						"Inputs":       inputs,
						"Action":       "/api/" + service + "/" + endpoint,
						"User":         user,
					})
					return
				}
				if r.Method == http.MethodPost {
					// Check endpoint scopes
					endpointKey := fmt.Sprintf("%s.%s", service, endpoint)
					if !checkEndpointScopes(w, r, endpointKey) {
						return
					}
					// Parse form values into a map
					var reqBody map[string]interface{}
					if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
						defer r.Body.Close()
						_ = json.NewDecoder(r.Body).Decode(&reqBody)
					} else {
						reqBody = map[string]interface{}{}
						_ = r.ParseForm()
						for k, v := range r.Form {
							if len(v) == 1 {
								if len(v[0]) == 0 {
									continue
								}
								reqBody[k] = v[0]
							} else {
								reqBody[k] = v
							}
						}
					}
					// For now, just echo the request body as JSON
					w.Header().Set("Content-Type", "application/json")
					b, _ := json.MarshalIndent(reqBody, "", "  ")
					w.Write(b)
					return
				}
			}
		}
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not found"))
	}))

	// Auth routes - only registered when auth is enabled
	if authEnabled {
		authMw := authRequired(storeInst)

		// loadEndpointScopes returns all stored endpoint scopes from the store
		loadEndpointScopes := func() map[string][]string {
			recs, _ := storeInst.Read("endpoint-scopes/", store.ReadPrefix())
			result := map[string][]string{}
			for _, rec := range recs {
				name := strings.TrimPrefix(rec.Key, "endpoint-scopes/")
				var scopes []string
				if err := json.Unmarshal(rec.Value, &scopes); err == nil && len(scopes) > 0 {
					result[name] = scopes
				}
			}
			return result
		}

		// Scopes management — per-endpoint scope requirements
		mux.HandleFunc("/auth/scopes", authMw(func(w http.ResponseWriter, r *http.Request) {
			userID := getUser(r)
			var user any
			if userID != "" {
				user = &TemplateUser{ID: userID}
			}
			success := false

			if r.Method == http.MethodPost {
				endpoint := r.FormValue("endpoint")
				scopesStr := r.FormValue("scopes")
				if endpoint != "" {
					if scopesStr == "" {
						_ = storeInst.Delete("endpoint-scopes/" + endpoint)
					} else {
						scopes := strings.Split(scopesStr, ",")
						for i := range scopes {
							scopes[i] = strings.TrimSpace(scopes[i])
						}
						b, _ := json.Marshal(scopes)
						_ = storeInst.Write(&store.Record{Key: "endpoint-scopes/" + endpoint, Value: b})
					}
					success = true
				}
			}

			// Discover endpoints
			services, _ := registry.ListServices()
			storedScopes := loadEndpointScopes()
			type endpointEntry struct {
				Name      string
				Service   string
				Endpoint  string
				Scopes    []string
				ScopesStr string
			}
			var endpoints []endpointEntry
			for _, svc := range services {
				fullSvcs, err := registry.GetService(svc.Name)
				if err != nil || len(fullSvcs) == 0 {
					continue
				}
				for _, ep := range fullSvcs[0].Endpoints {
					key := fmt.Sprintf("%s.%s", svc.Name, ep.Name)
					scopes := storedScopes[key]
					scopesStr := strings.Join(scopes, ", ")
					endpoints = append(endpoints, endpointEntry{
						Name:      key,
						Service:   svc.Name,
						Endpoint:  ep.Name,
						Scopes:    scopes,
						ScopesStr: scopesStr,
					})
				}
			}

			_ = renderPage(w, tmpls.scopes, map[string]any{
				"Title":     "Scopes",
				"Endpoints": endpoints,
				"User":      user,
				"Success":   success,
			})
		}))

		// Bulk set scopes for endpoints matching a pattern
		mux.HandleFunc("/auth/scopes/bulk", authMw(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Redirect(w, r, "/auth/scopes", http.StatusSeeOther)
				return
			}
			pattern := r.FormValue("pattern")
			scopesStr := r.FormValue("scopes")
			if pattern == "" {
				http.Redirect(w, r, "/auth/scopes", http.StatusSeeOther)
				return
			}
			scopes := []string{}
			if scopesStr != "" {
				scopes = strings.Split(scopesStr, ",")
				for i := range scopes {
					scopes[i] = strings.TrimSpace(scopes[i])
				}
			}

			// Find matching endpoints
			services, _ := registry.ListServices()
			for _, svc := range services {
				fullSvcs, err := registry.GetService(svc.Name)
				if err != nil || len(fullSvcs) == 0 {
					continue
				}
				for _, ep := range fullSvcs[0].Endpoints {
					key := fmt.Sprintf("%s.%s", svc.Name, ep.Name)
					matched := false
					if strings.HasSuffix(pattern, "*") {
						prefix := strings.TrimSuffix(pattern, "*")
						matched = strings.HasPrefix(key, prefix)
					} else {
						matched = key == pattern
					}
					if matched {
						if len(scopes) == 0 {
							_ = storeInst.Delete("endpoint-scopes/" + key)
						} else {
							b, _ := json.Marshal(scopes)
							_ = storeInst.Write(&store.Record{Key: "endpoint-scopes/" + key, Value: b})
						}
					}
				}
			}
			http.Redirect(w, r, "/auth/scopes", http.StatusSeeOther)
		}))

		mux.HandleFunc("/auth/logout", func(w http.ResponseWriter, r *http.Request) {
			http.SetCookie(w, &http.Cookie{Name: "micro_token", Value: "", Path: "/", Expires: time.Now().Add(-1 * time.Hour), HttpOnly: true})
			http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		})
		mux.HandleFunc("/auth/tokens", authMw(func(w http.ResponseWriter, r *http.Request) {
			userID := getUser(r)
			var user any
			if userID != "" {
				user = &TemplateUser{ID: userID}
			} else {
				user = nil
			}
			if r.Method == http.MethodPost {
				id := r.FormValue("id")
				typeStr := r.FormValue("type")
				scopesStr := r.FormValue("scopes")
				accType := "user"
				switch typeStr {
				case "admin":
					accType = "admin"
				case "service":
					accType = "service"
				}
				scopes := []string{"*"}
				if scopesStr != "" {
					scopes = strings.Split(scopesStr, ",")
					for i := range scopes {
						scopes[i] = strings.TrimSpace(scopes[i])
					}
				}
				acc := &Account{
					ID:       id,
					Type:     accType,
					Scopes:   scopes,
					Metadata: map[string]string{"created": time.Now().Format(time.RFC3339)},
				}
				// Service tokens do not require a password, generate a JWT directly
				tok, _ := GenerateJWT(acc.ID, acc.Type, acc.Scopes, 24*time.Hour)
				acc.Metadata["token"] = tok
				b, _ := json.Marshal(acc)
				_ = storeInst.Write(&store.Record{Key: "auth/" + id, Value: b})
				storeJWTToken(storeInst, tok, acc.ID) // Store the JWT token
				http.Redirect(w, r, "/auth/tokens", http.StatusSeeOther)
				return
			}
			recs, _ := storeInst.Read("auth/", store.ReadPrefix())
			var tokens []map[string]any
			for _, rec := range recs {
				var acc Account
				if err := json.Unmarshal(rec.Value, &acc); err == nil {
					tok := ""
					if t, ok := acc.Metadata["token"]; ok {
						tok = t
					}
					var tokenPrefix, tokenSuffix string
					if len(tok) > 12 {
						tokenPrefix = tok[:4]
						tokenSuffix = tok[len(tok)-4:]
					} else {
						tokenPrefix = tok
						tokenSuffix = ""
					}
					tokens = append(tokens, map[string]any{
						"ID":          acc.ID,
						"Type":        acc.Type,
						"Scopes":      acc.Scopes,
						"Metadata":    acc.Metadata,
						"Token":       tok,
						"TokenPrefix": tokenPrefix,
						"TokenSuffix": tokenSuffix,
					})
				}
			}
			_ = renderPage(w, tmpls.authTokens, map[string]any{"Title": "Tokens", "Tokens": tokens, "User": user, "Sub": userID})
		}))

		mux.HandleFunc("/auth/users", authMw(func(w http.ResponseWriter, r *http.Request) {
			userID := getUser(r)
			var user any
			if userID != "" {
				user = &TemplateUser{ID: userID}
			} else {
				user = nil
			}
			if r.Method == http.MethodPost {
				if del := r.FormValue("delete"); del != "" {
					// Delete user
					_ = storeInst.Delete("auth/" + del)
					deleteUserTokens(storeInst, del)
					// Mark default admin as deleted so it won't be recreated on restart
					if del == "admin" {
						_ = storeInst.Write(&store.Record{Key: "auth/.admin-deleted", Value: []byte("true")})
					}
					http.Redirect(w, r, "/auth/users", http.StatusSeeOther)
					return
				}
				id := r.FormValue("id")
				if id == "" {
					http.Redirect(w, r, "/auth/users", http.StatusSeeOther)
					return
				}
				pass := r.FormValue("password")
				typeStr := r.FormValue("type")
				accType := "user"
				if typeStr == "admin" {
					accType = "admin"
				}
				hash, _ := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
				acc := &Account{
					ID:       id,
					Type:     accType,
					Scopes:   []string{"*"},
					Metadata: map[string]string{"created": time.Now().Format(time.RFC3339), "password_hash": string(hash)},
				}
				b, _ := json.Marshal(acc)
				_ = storeInst.Write(&store.Record{Key: "auth/" + id, Value: b})
				http.Redirect(w, r, "/auth/users", http.StatusSeeOther)
				return
			}
			recs, _ := storeInst.Read("auth/", store.ReadPrefix())
			var users []Account
			for _, rec := range recs {
				var acc Account
				if err := json.Unmarshal(rec.Value, &acc); err == nil {
					if acc.Type == "user" || acc.Type == "admin" {
						users = append(users, acc)
					}
				}
			}
			_ = renderPage(w, tmpls.authUsers, map[string]any{"Title": "Users", "Users": users, "User": user})
		}))
		mux.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				loginTmpl, err := template.ParseFS(HTML, "web/templates/base.html", "web/templates/auth_login.html")
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("Template error: " + err.Error()))
					return
				}
				_ = loginTmpl.Execute(w, map[string]any{"Title": "Login", "Error": "", "User": getUser(r), "HideSidebar": true})
				return
			}
			if r.Method == http.MethodPost {
				id := r.FormValue("id")
				pass := r.FormValue("password")
				recKey := "auth/" + id
				recs, _ := storeInst.Read(recKey)
				if len(recs) == 0 {
					loginTmpl, _ := template.ParseFS(HTML, "web/templates/base.html", "web/templates/auth_login.html")
					_ = loginTmpl.Execute(w, map[string]any{"Title": "Login", "Error": "Invalid credentials", "User": "", "HideSidebar": true})
					return
				}
				var acc Account
				if err := json.Unmarshal(recs[0].Value, &acc); err != nil {
					loginTmpl, _ := template.ParseFS(HTML, "web/templates/base.html", "web/templates/auth_login.html")
					_ = loginTmpl.Execute(w, map[string]any{"Title": "Login", "Error": "Invalid credentials", "User": "", "HideSidebar": true})
					return
				}
				hash, ok := acc.Metadata["password_hash"]
				if !ok || bcrypt.CompareHashAndPassword([]byte(hash), []byte(pass)) != nil {
					loginTmpl, _ := template.ParseFS(HTML, "web/templates/base.html", "web/templates/auth_login.html")
					_ = loginTmpl.Execute(w, map[string]any{"Title": "Login", "Error": "Invalid credentials", "User": "", "HideSidebar": true})
					return
				}
				tok, err := GenerateJWT(acc.ID, acc.Type, acc.Scopes, 24*time.Hour)
				if err != nil {
					log.Printf("[LOGIN ERROR] Token generation failed: %v\nAccount: %+v", err, acc)
					loginTmpl, _ := template.ParseFS(HTML, "web/templates/base.html", "web/templates/auth_login.html")
					_ = loginTmpl.Execute(w, map[string]any{"Title": "Login", "Error": "Token error", "User": "", "HideSidebar": true})
					return
				}
				storeJWTToken(storeInst, tok, acc.ID) // Store the JWT token
				http.SetCookie(w, &http.Cookie{
					Name:     "micro_token",
					Value:    tok,
					Path:     "/",
					Expires:  time.Now().Add(time.Hour * 24),
					HttpOnly: true,
				})
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte("Method not allowed"))
		})
	} // end if authEnabled
}

func Run(c *cli.Context) error {
	addr := c.String("address")
	if addr == "" {
		addr = ":8080"
	}

	mcpAddr := c.String("mcp-address")

	// Auth follows the socket: on when the bind address is exposed, off on
	// loopback, overridable with --auth/--no-auth. When on, a machine token is
	// provisioned (supplied or generated); print a generated one once.
	authEnabled, genToken := ResolveAuth(c, addr)

	// Run the HTTP gateway (dashboard, REST, auth).
	opts := GatewayOptions{
		Address:     addr,
		AuthEnabled: authEnabled,
		Context:     c.Context,
	}

	if authEnabled {
		if genToken != "" {
			log.Printf("[auth] on (%s is exposed). Token (shown once): %s", addr, genToken)
		} else {
			log.Printf("[auth] on (%s is exposed). Using supplied MICRO_AUTH_TOKEN.", addr)
		}
	} else {
		log.Printf("[auth] off (%s is loopback). Scoped/paid tools still require a token.", addr)
	}

	// The MCP gateway runs independently of the HTTP API gateway, carrying the
	// production controls (rate limiting, scopes, auth, audit, circuit breaker,
	// x402 payments). Both gateways are started explicitly below and shut down
	// together when the first one exits or a signal arrives.
	var mcpServer *mcp.Server
	if mcpAddr != "" {
		mcpOpts, err := buildMCPOptions(c, mcpAddr)
		if err != nil {
			return err
		}
		mcpServer, err = mcp.NewServer(mcpOpts)
		if err != nil {
			return fmt.Errorf("failed to start MCP gateway: %w", err)
		}
		log.Printf("[mcp] gateway on %s", mcpAddr)
	}

	gw, err := StartGateway(opts)
	if err != nil {
		return fmt.Errorf("failed to start gateway: %w", err)
	}

	// Run both gateways until a signal or the first gateway exits.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	var eg errgroup.Group
	eg.Go(func() error { return gw.Wait() })
	if mcpServer != nil {
		eg.Go(func() error { return mcpServer.Serve() })
	}

	errCh := make(chan error, 1)
	go func() { errCh <- eg.Wait() }()

	select {
	case <-sigCh:
		log.Printf("[gateway] shutting down")
	case err = <-errCh:
		log.Printf("[gateway] gateway exited: %v", err)
	}

	_ = gw.Stop()
	if mcpServer != nil {
		_ = mcpServer.Stop()
	}
	return err
}

// buildMCPOptions assembles the MCP gateway options from the command flags. It
// carries the same production controls the standalone gateway binary had.
func buildMCPOptions(c *cli.Context, addr string) (mcp.Options, error) {
	logger := log.New(os.Stdout, "[mcp-gateway] ", log.LstdFlags)
	opts := mcp.Options{
		Registry: registry.DefaultRegistry,
		Address:  addr,
		Context:  c.Context,
		Logger:   logger,
	}

	// x402 payments: a config file (per-tool amounts) or the flags.
	if cfgPath := c.String("x402-config"); cfgPath != "" {
		cfg, err := x402.LoadConfig(cfgPath)
		if err != nil {
			return opts, fmt.Errorf("x402 config: %w", err)
		}
		opts.Payment = cfg
		logger.Printf("x402 payments enabled from %s (payTo %s)", cfgPath, cfg.PayTo)
	} else if payTo := c.String("x402-pay-to"); payTo != "" {
		opts.Payment = &x402.Config{
			PayTo:          payTo,
			Amount:         c.String("x402-amount"),
			Network:        c.String("x402-network"),
			FacilitatorURL: c.String("x402-facilitator"),
		}
	}

	if rps := c.Float64("rate-limit"); rps > 0 {
		opts.RateLimit = &mcp.RateLimitConfig{RequestsPerSecond: rps, Burst: c.Int("rate-burst")}
		logger.Printf("rate limit: %.0f req/s, burst %d", rps, c.Int("rate-burst"))
	}

	if c.Bool("auth") {
		opts.Auth = jwt.NewAuth()
		logger.Printf("JWT authentication enabled")
	}

	if scopes := c.StringSlice("scope"); len(scopes) > 0 {
		opts.Scopes = parseScopes(scopes)
	}

	if maxFail := c.Int("circuit-breaker"); maxFail > 0 {
		opts.CircuitBreaker = &mcp.CircuitBreakerConfig{
			MaxFailures: maxFail,
			Timeout:     c.Duration("circuit-breaker-timeout"),
		}
	}

	if c.Bool("audit") {
		opts.AuditFunc = func(r mcp.AuditRecord) {
			status := "ALLOWED"
			if !r.Allowed {
				status = "DENIED:" + r.DeniedReason
			}
			logger.Printf("[audit] %s tool=%s account=%s status=%s duration=%s",
				r.TraceID, r.Tool, r.AccountID, status, r.Duration)
		}
		logger.Printf("audit logging enabled")
	}

	return opts, nil
}

// parseScopes turns "tool=scope1,scope2" flag values into a tool→scopes map.
func parseScopes(raw []string) map[string][]string {
	scopes := make(map[string][]string)
	for _, s := range raw {
		parts := strings.SplitN(s, "=", 2)
		if len(parts) != 2 {
			continue
		}
		tool := strings.TrimSpace(parts[0])
		scopeList := strings.Split(parts[1], ",")
		for i := range scopeList {
			scopeList[i] = strings.TrimSpace(scopeList[i])
		}
		scopes[tool] = scopeList
	}
	return scopes
}

// mapGoTypeToJSON maps Go types to JSON schema types
func mapGoTypeToJSON(goType string) string {
	switch goType {
	case "string":
		return "string"
	case "int", "int32", "int64", "uint", "uint32", "uint64":
		return "integer"
	case "float32", "float64":
		return "number"
	case "bool":
		return "boolean"
	default:
		return "object"
	}
}

// --- PID FILES ---
func initAuth() error {
	// --- AUTH SETUP ---
	homeDir, _ := os.UserHomeDir()
	keyDir := filepath.Join(homeDir, "micro", "keys")
	privPath := filepath.Join(keyDir, "private.pem")
	pubPath := filepath.Join(keyDir, "public.pem")
	_ = os.MkdirAll(keyDir, 0700)
	// Generate keypair if not exist
	if _, err := os.Stat(privPath); os.IsNotExist(err) {
		priv, _ := rsa.GenerateKey(rand.Reader, 2048)
		privBytes := x509.MarshalPKCS1PrivateKey(priv)
		privPem := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes})
		_ = os.WriteFile(privPath, privPem, 0600)
		// Use PKIX format for public key
		pubBytes, _ := x509.MarshalPKIXPublicKey(&priv.PublicKey)
		pubPem := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})
		_ = os.WriteFile(pubPath, pubPem, 0644)
	}
	_, _ = os.ReadFile(privPath)
	_, _ = os.ReadFile(pubPath)
	// No hardcoded default credential. An admin account is created only when
	// the operator explicitly opts in via MICRO_ADMIN_PASSWORD (the docker-compose
	// file sets it to the documented admin/micro); otherwise the gateway runs on
	// the per-process machine token (see ResolveAuth) plus any accounts/tokens
	// created via /auth.
	return ensureAdminFromEnv(store.DefaultStore)
}

// ensureAdminFromEnv creates the operator-configured admin account when
// MICRO_ADMIN_PASSWORD is set. The default user is "admin" (override with
// MICRO_ADMIN_USER). Never overwrites an existing account, and an admin
// explicitly deleted via /auth/users stays deleted across restarts.
func ensureAdminFromEnv(storeInst store.Store) error {
	pass := os.Getenv("MICRO_ADMIN_PASSWORD")
	if pass == "" {
		return nil
	}
	adminID := os.Getenv("MICRO_ADMIN_USER")
	if adminID == "" {
		adminID = "admin"
	}
	adminKey := "auth/" + adminID
	if delRecs, _ := storeInst.Read("auth/.admin-deleted"); len(delRecs) > 0 && adminID == "admin" {
		return nil
	}
	if recs, _ := storeInst.Read(adminKey); len(recs) > 0 {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	acc := &Account{
		ID:       adminID,
		Type:     "admin",
		Scopes:   []string{"*"},
		Metadata: map[string]string{"created": time.Now().Format(time.RFC3339), "password_hash": string(hash)},
	}
	b, _ := json.Marshal(acc)
	_ = storeInst.Write(&store.Record{Key: adminKey, Value: b})
	log.Printf("[auth] created admin account %q from MICRO_ADMIN_PASSWORD", adminID)
	return nil
}

// parseStartTime parses a string as RFC3339 time
func parseStartTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

// gatewayFlags are shared by `micro gateway` and its deprecated `server` alias.
func gatewayFlags() []cli.Flag {
	flags := []cli.Flag{
		&cli.StringFlag{
			Name:    "address",
			Usage:   "HTTP address for the dashboard/API",
			EnvVars: []string{"MICRO_SERVER_ADDRESS"},
			Value:   ":8080",
		},
		&cli.StringFlag{
			Name:    "mcp-address",
			Usage:   "MCP protocol address (e.g., :3000). Enables the MCP gateway for AI tools.",
			EnvVars: []string{"MICRO_MCP_ADDRESS"},
		},
		&cli.Float64Flag{
			Name:    "rate-limit",
			Usage:   "MCP: requests per second per tool (0 = unlimited)",
			EnvVars: []string{"MCP_RATE_LIMIT"},
		},
		&cli.IntFlag{
			Name:    "rate-burst",
			Usage:   "MCP: rate limit burst size",
			Value:   20,
			EnvVars: []string{"MCP_RATE_BURST"},
		},
		&cli.BoolFlag{
			Name:    "audit",
			Usage:   "MCP: enable audit logging to stdout",
			EnvVars: []string{"MCP_AUDIT"},
		},
		&cli.StringSliceFlag{
			Name:  "scope",
			Usage: "MCP: tool scope requirement (format: tool=scope1,scope2)",
		},
		&cli.IntFlag{
			Name:    "circuit-breaker",
			Usage:   "MCP: circuit breaker max failures before opening (0 = disabled)",
			EnvVars: []string{"MCP_CIRCUIT_BREAKER"},
		},
		&cli.DurationFlag{
			Name:    "circuit-breaker-timeout",
			Usage:   "MCP: circuit breaker open-state timeout before half-open probe",
			Value:   30 * time.Second,
			EnvVars: []string{"MCP_CIRCUIT_BREAKER_TIMEOUT"},
		},
		&cli.StringFlag{
			Name:    "x402-pay-to",
			Usage:   "MCP: enable x402 payments for tool calls; the address payments are sent to",
			EnvVars: []string{"X402_PAY_TO"},
		},
		&cli.StringFlag{
			Name:    "x402-amount",
			Usage:   "MCP: default amount required per tool call, in the asset's smallest unit (e.g. 10000 = 0.01 USDC)",
			EnvVars: []string{"X402_AMOUNT"},
		},
		&cli.StringFlag{
			Name:    "x402-network",
			Usage:   "MCP: payment network: base (default), solana, ...",
			Value:   "base",
			EnvVars: []string{"X402_NETWORK"},
		},
		&cli.StringFlag{
			Name:    "x402-facilitator",
			Usage:   "MCP: x402 facilitator URL (Coinbase CDP, Alchemy, or self-hosted)",
			EnvVars: []string{"X402_FACILITATOR"},
		},
		&cli.StringFlag{
			Name:    "x402-config",
			Usage:   "MCP: path to an x402 config file; overrides the x402-* flags",
			EnvVars: []string{"X402_CONFIG"},
		},
	}
	return append(flags, AuthFlags()...)
}

func init() {
	cmd.Register(&cli.Command{
		Name:  "gateway",
		Usage: "Run the gateway: HTTP API, dashboard, auth, and MCP tools for your services",
		Description: `Run the gateway in front of your running services.

It serves an HTTP API and dashboard (with auth) on --address, and — when
--mcp-address is set — an MCP gateway that exposes every discovered service as
an AI-callable tool, with optional rate limiting, scopes, JWT auth, audit
logging, circuit breaking, and x402 payments.

Services are discovered through the registry (select it with the global
--registry / --registry_address flags). This is the gateway you deploy in
production; for the local development loop use ` + "`micro run`" + `.

Examples:
  micro gateway                                 # dashboard/API on :8080
  micro gateway --mcp-address :3000             # + MCP gateway on :3000
  micro gateway --mcp-address :3000 --auth --audit --rate-limit 100
  micro gateway --registry consul --registry_address consul:8500 --mcp-address :3000`,
		Action: Run,
		Flags:  gatewayFlags(),
	})

	// Deprecated alias: `micro server` still works but is hidden and warns.
	cmd.Register(&cli.Command{
		Name:   "server",
		Hidden: true,
		Usage:  "Deprecated alias for `micro gateway`",
		Action: func(c *cli.Context) error {
			fmt.Fprintln(os.Stderr, "warning: `micro server` is deprecated and will be removed; use `micro gateway`")
			return Run(c)
		},
		Flags: gatewayFlags(),
	})
}
