// Package app defines a human-facing application backed by service capabilities.
// Apps pair a portable definition with an fs.FS of public assets. They can be
// served by net/http or web.Service; generation, storage and host policy remain
// with the caller.
package app

import (
	"bytes"
	"fmt"
	"io/fs"
	"net/http"
	"regexp"
	"strings"

	"go-micro.dev/v6/registry"
)

// Dependency declares service endpoints an app expects. It grants no access.
// Endpoints use RPC names such as "Notes.List". An empty list requires only
// the named service. Hosts provide the actual authenticated API connection.
type Dependency struct {
	Name      string   `json:"name"`
	Endpoints []string `json:"endpoints,omitempty"`
}

// Definition is the portable description of one app version. Entrypoint is a
// file path relative to the asset filesystem, for example "index.html".
// Version is an opaque release identifier; this package does not manage revisions.
type Definition struct {
	Name        string       `json:"name"`
	Version     string       `json:"version"`
	Description string       `json:"description,omitempty"`
	Entrypoint  string       `json:"entrypoint"`
	Services    []Dependency `json:"services,omitempty"`
}

var identifier = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// Validate checks the definition without requiring assets or a running registry.
func (d Definition) Validate() error {
	if !identifier.MatchString(d.Name) || !identifier.MatchString(d.Version) {
		return fmt.Errorf("app: name and version must be nonempty identifiers (letters, digits, dot, dash or underscore)")
	}
	if !publicPath(d.Entrypoint) {
		return fmt.Errorf("app: entrypoint must name a public file relative to the asset root")
	}
	seen := make(map[string]bool)
	for _, dep := range d.Services {
		if !identifier.MatchString(dep.Name) || seen[dep.Name] {
			return fmt.Errorf("app: invalid or duplicate service %q", dep.Name)
		}
		seen[dep.Name] = true
		endpoints := make(map[string]bool)
		for _, endpoint := range dep.Endpoints {
			if strings.TrimSpace(endpoint) != endpoint || endpoint == "" || endpoints[endpoint] {
				return fmt.Errorf("app: invalid or duplicate endpoint %q for %s", endpoint, dep.Name)
			}
			endpoints[endpoint] = true
		}
	}
	return nil
}

// App is a validated definition and its public assets. Treat the supplied fs.FS
// as immutable for the lifetime of the App. Supply only files intended for public
// serving; this handler is not a sandbox for untrusted code or filesystem links.
type App struct {
	definition Definition
	assets     fs.FS
}

// New validates an app and verifies its entrypoint exists as a regular file.
func New(def Definition, assets fs.FS) (*App, error) {
	if err := def.Validate(); err != nil {
		return nil, err
	}
	if assets == nil {
		return nil, fmt.Errorf("app: assets are required")
	}
	info, err := fs.Stat(assets, def.Entrypoint)
	if err != nil {
		return nil, fmt.Errorf("app: entrypoint: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("app: entrypoint must be a regular file")
	}
	return &App{definition: clone(def), assets: assets}, nil
}

// Definition returns an independent copy that hosts can serialize or advertise.
func (a *App) Definition() Definition { return clone(a.definition) }

// Handler serves the entrypoint at / and public files at their relative paths.
// Missing paths return 404; directories are never listed. Mount beneath a prefix
// with http.StripPrefix. The host owns middleware, API routes and authentication.
func (a *App) Handler() http.Handler { return http.HandlerFunc(a.serveHTTP) }

func (a *App) serveHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/")
	if name == "" {
		name = a.definition.Entrypoint
	}
	if !publicPath(name) {
		http.NotFound(w, r)
		return
	}
	info, err := fs.Stat(a.assets, name)
	if err != nil || !info.Mode().IsRegular() {
		http.NotFound(w, r)
		return
	}
	data, err := fs.ReadFile(a.assets, name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, name, info.ModTime(), bytes.NewReader(data))
}

func publicPath(name string) bool {
	if !fs.ValidPath(name) || name == "." || strings.Contains(name, `\`) {
		return false
	}
	for _, part := range strings.Split(name, "/") {
		if strings.HasPrefix(part, ".") {
			return false
		}
	}
	return true
}

// CheckDependencies checks registry metadata at a point in time. At least one
// registered version of each service must declare all requested endpoints.
// It neither probes health nor checks caller permissions or schema compatibility.
func (a *App) CheckDependencies(reg registry.Registry) error {
	if reg == nil {
		return fmt.Errorf("app: registry is required")
	}
	for _, dep := range a.definition.Services {
		services, err := reg.GetService(dep.Name)
		if err != nil {
			return fmt.Errorf("app: service %s: %w", dep.Name, err)
		}
		found := false
		for _, svc := range services {
			if svc == nil || len(svc.Nodes) == 0 {
				continue
			}
			endpoints := make(map[string]bool)
			for _, ep := range svc.Endpoints {
				if ep != nil {
					endpoints[ep.Name] = true
				}
			}
			complete := true
			for _, endpoint := range dep.Endpoints {
				complete = complete && endpoints[endpoint]
			}
			found = found || complete
		}
		if !found {
			return fmt.Errorf("app: service %s has no registered instance with required endpoints %v", dep.Name, dep.Endpoints)
		}
	}
	return nil
}

func clone(d Definition) Definition {
	d.Services = append([]Dependency(nil), d.Services...)
	for i := range d.Services {
		d.Services[i].Endpoints = append([]string(nil), d.Services[i].Endpoints...)
	}
	return d
}
