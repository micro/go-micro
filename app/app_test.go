package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"go-micro.dev/v6/registry"
)

func TestDefinitionRoundTripAndIsolation(t *testing.T) {
	def := Definition{Name: "notes", Version: "1.0", Entrypoint: "index.html", Services: []Dependency{{Name: "notes", Endpoints: []string{"Notes.List"}}}}
	data, err := json.Marshal(def)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Definition
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	a, err := New(decoded, fstest.MapFS{"index.html": {Data: []byte("hello")}})
	if err != nil {
		t.Fatal(err)
	}
	decoded.Services[0].Endpoints[0] = "changed"
	copy := a.Definition()
	copy.Services[0].Endpoints[0] = "changed again"
	if got := a.Definition().Services[0].Endpoints[0]; got != "Notes.List" {
		t.Fatal(got)
	}
}

func TestAssets(t *testing.T) {
	a, err := New(Definition{Name: "notes", Version: "1", Entrypoint: "start.html"}, fstest.MapFS{
		"start.html":      {Data: []byte("<h1>Notes</h1>")},
		"style.css":       {Data: []byte("body {color: black}")},
		"private/.key":    {Data: []byte("secret")},
		"folder/file.txt": {Data: []byte("file")},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{"GET", "/", 200}, {"HEAD", "/", 200}, {"GET", "/style.css", 200},
		{"GET", "/folder/", 404}, {"GET", "/missing", 404}, {"GET", "/../start.html", 404},
		{"GET", "/%2e%2e/start.html", 404}, {"GET", "/private/.key", 404},
		{"GET", "//start.html", 404}, {"POST", "/", 405},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			a.Handler().ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
			if w.Code != tc.status {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			if tc.method == "HEAD" && w.Body.Len() != 0 {
				t.Fatal("HEAD returned a body")
			}
		})
	}
	w := httptest.NewRecorder()
	http.StripPrefix("/apps/notes", a.Handler()).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/apps/notes/", nil))
	if !strings.Contains(w.Body.String(), "<h1>Notes</h1>") {
		t.Fatal(w.Body.String())
	}
}

func TestInvalidDefinitions(t *testing.T) {
	for _, entry := range []string{"", ".", "../index.html", "/index.html", "dir/.secret", `dir\index.html`, "missing.html", "dir"} {
		_, err := New(Definition{Name: "notes", Version: "1", Entrypoint: entry}, fstest.MapFS{"dir/a.html": {Data: []byte("ok")}})
		if err == nil {
			t.Errorf("accepted %q", entry)
		}
	}
	for _, def := range []Definition{
		{Name: "bad/name", Version: "1", Entrypoint: "index.html"},
		{Name: "notes", Entrypoint: "index.html"},
		{Name: "notes", Version: "1", Entrypoint: "index.html", Services: []Dependency{{Name: "x"}, {Name: "x"}}},
		{Name: "notes", Version: "1", Entrypoint: "index.html", Services: []Dependency{{Name: "x", Endpoints: []string{""}}}},
	} {
		if def.Validate() == nil {
			t.Errorf("accepted %+v", def)
		}
	}
}

func TestDependenciesRequireOneCompleteVersion(t *testing.T) {
	a, err := New(Definition{Name: "notes", Version: "1", Entrypoint: "index.html", Services: []Dependency{{Name: "notes", Endpoints: []string{"Notes.List", "Notes.Add"}}}}, fstest.MapFS{"index.html": {Data: []byte("ok")}})
	if err != nil {
		t.Fatal(err)
	}
	r := registry.NewMemoryRegistry()
	if a.CheckDependencies(r) == nil {
		t.Fatal("missing service accepted")
	}
	for _, v := range []struct{ version, endpoint string }{{"1", "Notes.List"}, {"2", "Notes.Add"}} {
		if err := r.Register(&registry.Service{Name: "notes", Version: v.version, Nodes: []*registry.Node{{Id: v.version, Address: "127.0.0.1:1"}}, Endpoints: []*registry.Endpoint{{Name: v.endpoint}}}); err != nil {
			t.Fatal(err)
		}
	}
	if a.CheckDependencies(r) == nil {
		t.Fatal("incompatible versions combined")
	}
	if err := r.Register(&registry.Service{Name: "notes", Version: "3", Nodes: []*registry.Node{{Id: "3", Address: "127.0.0.1:1"}}, Endpoints: []*registry.Endpoint{{Name: "Notes.List"}, {Name: "Notes.Add"}}}); err != nil {
		t.Fatal(err)
	}
	if err := a.CheckDependencies(r); err != nil {
		t.Fatal(err)
	}
}
