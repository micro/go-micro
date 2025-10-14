package server

import (
	"bytes"
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
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"text/template"
	"time"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v5/cmd"
	"go-micro.dev/v5/registry"
	"go-micro.dev/v5/store"
	"golang.org/x/crypto/bcrypt"
)

// HTML is the embedded filesystem for templates and static files, set by main.go
var HTML fs.FS

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
}

type TemplateUser struct {
	ID string
}

// Define a local Account struct to replace auth.Account
// (matches fields used in the code)
type Account struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"`
	Scopes   []string          `json:"scopes"`
	Metadata map[string]string `json:"metadata"`
}

func parseTemplates() *templates {
	return &templates{
		api:        template.Must(template.ParseFS(HTML, "html/templates/base.html", "html/templates/api.html")),
		service:    template.Must(template.ParseFS(HTML, "html/templates/base.html", "html/templates/service.html")),
		form:       template.Must(template.ParseFS(HTML, "html/templates/base.html", "html/templates/form.html")),
		home:       template.Must(template.ParseFS(HTML, "html/templates/base.html", "html/templates/home.html")),
		logs:       template.Must(template.ParseFS(HTML, "html/templates/base.html", "html/templates/logs.html")),
		log:        template.Must(template.ParseFS(HTML, "html/templates/base.html", "html/templates/log.html")),
		status:     template.Must(template.ParseFS(HTML, "html/templates/base.html", "html/templates/status.html")),
		authTokens: template.Must(template.ParseFS(HTML, "html/templates/base.html", "html/templates/auth_tokens.html")),
		authLogin:  template.Must(template.ParseFS(HTML, "html/templates/base.html", "html/templates/auth_login.html")),
		authUsers:  template.Must(template.ParseFS(HTML, "html/templates/base.html", "html/templates/auth_users.html")),
	}
}

// Helper to render templates
func render(w http.ResponseWriter, tmpl *template.Template, data any) error {
	return tmpl.Execute(w, data)
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
	storeInst.Write(&store.Record{Key: "jwt/" + token, Value: []byte(userID)})
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
			storeInst.Delete(rec.Key)
		}
	}
}

// Updated authRequired to accept storeInst as argument
func authRequired(storeInst store.Store) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			var token string
			// 1. Check Authorization: Bearer header
			authz := r.Header.Get("Authorization")
			if strings.HasPrefix(authz, "Bearer ") {
				token = strings.TrimPrefix(authz, "Bearer ")
				token = strings.TrimSpace(token)
			}
			// 2. Fallback to micro_token cookie if no header
			if token == "" {
				cookie, err := r.Cookie("micro_token")
				if err == nil && cookie.Value != "" {
					token = cookie.Value
				}
			}
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
		pid := "-"
		if len(lines) > 0 && len(lines[0]) > 0 {
			pid = lines[0]
		}
		serviceCount++
		if pid != "-" {
			if _, err := os.FindProcess(parsePid(pid)); err == nil {
				if processRunning(pid) {
					runningCount++
				} else {
					stoppedCount++
				}
			} else {
				stoppedCount++
			}
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

func getSidebarEndpoints() ([]map[string]string, error) {
	apiCache.Lock()
	defer apiCache.Unlock()
	if apiCache.data != nil && time.Since(apiCache.time) < 30*time.Second {
		if v, ok := apiCache.data["SidebarEndpoints"]; ok {
			if endpoints, ok := v.([]map[string]string); ok {
				return endpoints, nil
			}
		}
	}
	services, err := registry.ListServices()
	if err != nil {
		return nil, err
	}
	var sidebarEndpoints []map[string]string
	for _, srv := range services {
		anchor := strings.ReplaceAll(srv.Name, ".", "-")
		sidebarEndpoints = append(sidebarEndpoints, map[string]string{"Name": srv.Name, "Anchor": anchor})
	}
	sort.Slice(sidebarEndpoints, func(i, j int) bool {
		return sidebarEndpoints[i]["Name"] < sidebarEndpoints[j]["Name"]
	})
	return sidebarEndpoints, nil
}

func registerHandlers(tmpls *templates, storeInst store.Store) {
	authMw := authRequired(storeInst)
	wrap := wrapAuth(authMw)

	// Serve static files from root (not /html/) with correct Content-Type
	http.HandleFunc("/styles.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		f, err := HTML.Open("html/styles.css")
		if err != nil {
			w.WriteHeader(404)
			return
		}
		defer f.Close()
		io.Copy(w, f)
	})

	http.HandleFunc("/main.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		f, err := HTML.Open("html/main.js")
		if err != nil {
			w.WriteHeader(404)
			return
		}
		defer f.Close()
		io.Copy(w, f)
	})

	// Serve /html/styles.css and /html/main.js for compatibility
	http.HandleFunc("/html/styles.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		f, err := HTML.Open("html/styles.css")
		if err != nil {
			w.WriteHeader(404)
			return
		}
		defer f.Close()
		io.Copy(w, f)
	})
	http.HandleFunc("/html/main.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		f, err := HTML.Open("html/main.js")
		if err != nil {
			w.WriteHeader(404)
			return
		}
		defer f.Close()
		io.Copy(w, f)
	})

	http.HandleFunc("/", wrap(func(w http.ResponseWriter, r *http.Request) {
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
			// Do NOT include SidebarEndpoints on home page
			err := tmpls.home.Execute(w, map[string]any{
				"Title":        "Micro Dashboard",
				"WebLink":      "/",
				"ServiceCount": serviceCount,
				"RunningCount": runningCount,
				"StoppedCount": stoppedCount,
				"StatusDot":    statusDot,
				"User":         user,
				// No SidebarEndpoints or SidebarEndpointsEnabled here
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
						apiPath := fmt.Sprintf("/api/%s/%s/%s", s.Name, parts[0], parts[1])
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
			// Add API auth doc at the top
			apiData["ApiAuthDoc"] = `<div style='background:#f8f8e8; border:1px solid #e0e0b0; padding:1em; margin-bottom:2em; font-size:1.08em;'>
<b>API Authentication Required:</b> All API calls to <code>/api/...</code> endpoints (except this page) must include an <b>Authorization: Bearer &lt;token&gt;</b> header. <br>
You can generate tokens on the <a href='/auth/tokens'>Tokens page</a>.
</div>`
			_ = render(w, tmpls.api, apiData)
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
			_ = render(w, tmpls.service, map[string]any{"Title": "Services", "WebLink": "/", "Services": serviceNames, "User": user})
			return
		}
		if path == "/logs" || path == "/logs/" {
			// Do NOT include SidebarEndpoints on this page
			homeDir, err := os.UserHomeDir()
			if err != nil {
				w.WriteHeader(500)
				w.Write([]byte("Could not get home directory"))
				return
			}
			logsDir := homeDir + "/micro/logs"
			dirEntries, err := os.ReadDir(logsDir)
			if err != nil {
				w.WriteHeader(500)
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
			_ = render(w, tmpls.logs, map[string]any{"Title": "Logs", "WebLink": "/", "Services": serviceNames, "User": user})
			return
		}
		if strings.HasPrefix(path, "/logs/") {
			// Do NOT include SidebarEndpoints on this page
			service := strings.TrimPrefix(path, "/logs/")
			if service == "" {
				w.WriteHeader(404)
				w.Write([]byte("Service not specified"))
				return
			}
			homeDir, err := os.UserHomeDir()
			if err != nil {
				w.WriteHeader(500)
				w.Write([]byte("Could not get home directory"))
				return
			}
			logFilePath := homeDir + "/micro/logs/" + service + ".log"
			f, err := os.Open(logFilePath)
			if err != nil {
				w.WriteHeader(404)
				w.Write([]byte("Could not open log file for service: " + service))
				return
			}
			defer f.Close()
			logBytes, err := io.ReadAll(f)
			if err != nil {
				w.WriteHeader(500)
				w.Write([]byte("Could not read log file for service: " + service))
				return
			}
			logText := string(logBytes)
			_ = render(w, tmpls.log, map[string]any{"Title": "Logs for " + service, "WebLink": "/logs", "Service": service, "Log": logText, "User": user})
			return
		}
		if path == "/status" {
			// Do NOT include SidebarEndpoints on this page
			homeDir, err := os.UserHomeDir()
			if err != nil {
				w.WriteHeader(500)
				w.Write([]byte("Could not get home directory"))
				return
			}
			pidDir := homeDir + "/micro/run"
			dirEntries, err := os.ReadDir(pidDir)
			if err != nil {
				w.WriteHeader(500)
				w.Write([]byte("Could not list pid directory: " + err.Error()))
				return
			}
			statuses := []map[string]string{}
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
				status := "stopped"
				if pid != "-" {
					if _, err := os.FindProcess(parsePid(pid)); err == nil {
						if processRunning(pid) {
							status = "running"
						}
					} else {
						status = "stopped"
					}
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
			_ = render(w, tmpls.status, map[string]any{"Title": "Service Status", "WebLink": "/", "Statuses": statuses, "User": user})
			return
		}
		// Match /{service} and /{service}/{endpoint}
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) >= 1 && parts[0] != "api" && parts[0] != "html" && parts[0] != "services" {
			service := parts[0]
			if len(parts) == 1 {
				s, err := registry.GetService(service)
				if err != nil || len(s) == 0 {
					w.WriteHeader(404)
					w.Write([]byte(fmt.Sprintf("Service not found: %s", service)))
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
				_ = render(w, tmpls.service, map[string]any{
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
					w.WriteHeader(404)
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
					w.WriteHeader(404)
					w.Write([]byte("Endpoint not found"))
					return
				}
				if r.Method == "GET" {
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
					_ = render(w, tmpls.form, map[string]any{
						"Title":        "Service: " + service,
						"WebLink":      "/",
						"ServiceName":  service,
						"EndpointName": ep.Name,
						"Inputs":       inputs,
						"Action":       service + "/" + endpoint,
						"User":         user,
					})
					return
				}
				if r.Method == "POST" {
					// Parse form values into a map
					var reqBody map[string]interface{}
					if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
						defer r.Body.Close()
						json.NewDecoder(r.Body).Decode(&reqBody)
					} else {
						reqBody = map[string]interface{}{}
						r.ParseForm()
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
		w.WriteHeader(404)
		w.Write([]byte("Not found"))
	}))
	http.HandleFunc("/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "micro_token", Value: "", Path: "/", Expires: time.Now().Add(-1 * time.Hour), HttpOnly: true})
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
	})
	http.HandleFunc("/auth/tokens", authMw(func(w http.ResponseWriter, r *http.Request) {
		userID := getUser(r)
		var user any
		if userID != "" {
			user = &TemplateUser{ID: userID}
		} else {
			user = nil
		}
		if r.Method == "POST" {
			id := r.FormValue("id")
			typeStr := r.FormValue("type")
			scopesStr := r.FormValue("scopes")
			accType := "user"
			if typeStr == "admin" {
				accType = "admin"
			} else if typeStr == "service" {
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
			storeInst.Write(&store.Record{Key: "auth/" + id, Value: b})
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
		_ = tmpls.authTokens.Execute(w, map[string]any{"Title": "Auth Tokens", "Tokens": tokens, "User": user, "Sub": userID})
	}))

	http.HandleFunc("/auth/users", authMw(func(w http.ResponseWriter, r *http.Request) {
		userID := getUser(r)
		var user any
		if userID != "" {
			user = &TemplateUser{ID: userID}
		} else {
			user = nil
		}
		if r.Method == "POST" {
			if del := r.FormValue("delete"); del != "" {
				// Delete user
				storeInst.Delete("auth/" + del)
				deleteUserTokens(storeInst, del) // Delete all JWT tokens for this user
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
			storeInst.Write(&store.Record{Key: "auth/" + id, Value: b})
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
		_ = tmpls.authUsers.Execute(w, map[string]any{"Title": "User Accounts", "Users": users, "User": user})
	}))
	http.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			loginTmpl, err := template.ParseFS(HTML, "html/templates/base.html", "html/templates/auth_login.html")
			if err != nil {
				w.WriteHeader(500)
				w.Write([]byte("Template error: " + err.Error()))
				return
			}
			_ = loginTmpl.Execute(w, map[string]any{"Title": "Login", "Error": "", "User": getUser(r), "HideSidebar": true})
			return
		}
		if r.Method == "POST" {
			id := r.FormValue("id")
			pass := r.FormValue("password")
			recKey := "auth/" + id
			recs, _ := storeInst.Read(recKey)
			if len(recs) == 0 {
				loginTmpl, _ := template.ParseFS(HTML, "html/templates/base.html", "html/templates/auth_login.html")
				_ = loginTmpl.Execute(w, map[string]any{"Title": "Login", "Error": "Invalid credentials", "User": "", "HideSidebar": true})
				return
			}
			var acc Account
			if err := json.Unmarshal(recs[0].Value, &acc); err != nil {
				loginTmpl, _ := template.ParseFS(HTML, "html/templates/base.html", "html/templates/auth_login.html")
				_ = loginTmpl.Execute(w, map[string]any{"Title": "Login", "Error": "Invalid credentials", "User": "", "HideSidebar": true})
				return
			}
			hash, ok := acc.Metadata["password_hash"]
			if !ok || bcrypt.CompareHashAndPassword([]byte(hash), []byte(pass)) != nil {
				loginTmpl, _ := template.ParseFS(HTML, "html/templates/base.html", "html/templates/auth_login.html")
				_ = loginTmpl.Execute(w, map[string]any{"Title": "Login", "Error": "Invalid credentials", "User": "", "HideSidebar": true})
				return
			}
			tok, err := GenerateJWT(acc.ID, acc.Type, acc.Scopes, 24*time.Hour)
			if err != nil {
				log.Printf("[LOGIN ERROR] Token generation failed: %v\nAccount: %+v", err, acc)
				loginTmpl, _ := template.ParseFS(HTML, "html/templates/base.html", "html/templates/auth_login.html")
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
		w.WriteHeader(405)
		w.Write([]byte("Method not allowed"))
	})
}

func Run(c *cli.Context) error {
	if err := initAuth(); err != nil {
		log.Fatalf("Failed to initialize auth: %v", err)
	}
	homeDir, _ := os.UserHomeDir()
	keyDir := filepath.Join(homeDir, "micro", "keys")
	privPath := filepath.Join(keyDir, "private.pem")
	pubPath := filepath.Join(keyDir, "public.pem")
	if err := InitJWTKeys(privPath, pubPath); err != nil {
		log.Fatalf("Failed to init JWT keys: %v", err)
	}
	storeInst := store.DefaultStore
	tmpls := parseTemplates()
	registerHandlers(tmpls, storeInst)
	addr := c.String("address")
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("[micro-server] Web/API listening on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Web/API server error: %v", err)
	}
	return nil
}

// --- PID FILES ---
// --- PID FILES ---
func parsePid(pidStr string) int {
	pid, _ := strconv.Atoi(pidStr)
	return pid
}
func processRunning(pid string) bool {
	proc, err := os.FindProcess(parsePid(pid))
	if err != nil {
		return false
	}
	// On unix, sending syscall.Signal(0) checks if process exists
	return proc.Signal(syscall.Signal(0)) == nil
}

func generateKeyPair(bits int) (*rsa.PrivateKey, error) {
	priv, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, err
	}
	return priv, nil
}
func exportPrivateKeyAsPEM(priv *rsa.PrivateKey) ([]byte, error) {
	privKeyBytes := x509.MarshalPKCS1PrivateKey(priv)
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privKeyBytes,
	}
	var buf bytes.Buffer
	err := pem.Encode(&buf, block)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
func exportPublicKeyAsPEM(pub *rsa.PublicKey) ([]byte, error) {
	pubKeyBytes := x509.MarshalPKCS1PublicKey(pub)
	block := &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: pubKeyBytes,
	}
	var buf bytes.Buffer
	err := pem.Encode(&buf, block)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
func importPrivateKeyFromPEM(privKeyPEM []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(privKeyPEM)
	if block == nil {
		return nil, fmt.Errorf("invalid PEM block")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}
func importPublicKeyFromPEM(pubKeyPEM []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pubKeyPEM)
	if block == nil {
		return nil, fmt.Errorf("invalid PEM block")
	}
	return x509.ParsePKCS1PublicKey(block.Bytes)
}
func initAuth() error {
	// --- AUTH SETUP ---
	homeDir, _ := os.UserHomeDir()
	keyDir := filepath.Join(homeDir, "micro", "keys")
	privPath := filepath.Join(keyDir, "private.pem")
	pubPath := filepath.Join(keyDir, "public.pem")
	os.MkdirAll(keyDir, 0700)
	// Generate keypair if not exist
	if _, err := os.Stat(privPath); os.IsNotExist(err) {
		priv, _ := rsa.GenerateKey(rand.Reader, 2048)
		privBytes := x509.MarshalPKCS1PrivateKey(priv)
		privPem := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes})
		os.WriteFile(privPath, privPem, 0600)
		// Use PKIX format for public key
		pubBytes, _ := x509.MarshalPKIXPublicKey(&priv.PublicKey)
		pubPem := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})
		os.WriteFile(pubPath, pubPem, 0644)
	}
	_, _ = os.ReadFile(privPath)
	_, _ = os.ReadFile(pubPath)
	storeInst := store.DefaultStore
	// --- Ensure default admin account exists ---
	adminID := "admin"
	adminPass := "micro"
	adminKey := "auth/" + adminID
	if recs, _ := storeInst.Read(adminKey); len(recs) == 0 {
		// Hash the admin password with bcrypt
		hash, err := bcrypt.GenerateFromPassword([]byte(adminPass), bcrypt.DefaultCost)
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
		storeInst.Write(&store.Record{Key: adminKey, Value: b})
	}
	return nil
}

// parseStartTime parses a string as RFC3339 time
func parseStartTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}
func init() {
	cmd.Register(&cli.Command{
		Name:   "server",
		Usage:  "Run the micro server",
		Action: Run,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "address",
				Usage:   "Address to listen on",
				EnvVars: []string{"MICRO_SERVER_ADDRESS"},
				Value:   ":8080",
			},
		},
	})
}
