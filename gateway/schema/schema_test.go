package schema

import (
	"context"
	"testing"
	"time"

	"go-micro.dev/v6/registry"
)

func TestResolverDiscoversEndpoints(t *testing.T) {
	reg := registry.NewMemoryRegistry()
	svc := &registry.Service{
		Name: "blog",
		Nodes: []*registry.Node{{
			Id:      "blog-1",
			Address: "localhost:9090",
		}},
		Endpoints: []*registry.Endpoint{
			{
				Name: "Blog.Create",
				Metadata: map[string]string{
					"description": "Create a blog post",
					"scopes":      "blog:write, blog:admin",
					"example":     `{"title":"hi"}`,
				},
				Request: &registry.Value{
					Name: "Request",
					Values: []*registry.Value{
						{Name: "title", Type: "string"},
						{Name: "likes", Type: "int64"},
					},
				},
				Response: &registry.Value{
					Name: "Response",
					Values: []*registry.Value{
						{Name: "id", Type: "string"},
					},
				},
			},
			{Name: "Blog.Read"},
		},
	}
	if err := reg.Register(svc); err != nil {
		t.Fatal(err)
	}

	r := New(reg)
	if _, err := r.Refresh(); err != nil {
		t.Fatal(err)
	}

	eps := r.Endpoints()
	if len(eps) != 2 {
		t.Fatalf("got %d endpoints, want 2", len(eps))
	}

	create, ok := r.Endpoint("blog.Blog.Create")
	if !ok {
		t.Fatal("expected blog.Blog.Create")
	}
	if create.Service != "blog" || create.Method != "Blog.Create" {
		t.Errorf("unexpected service/method: %s/%s", create.Service, create.Method)
	}
	if create.Description != "Create a blog post" {
		t.Errorf("description = %q", create.Description)
	}
	if len(create.Scopes) != 2 || create.Scopes[0] != "blog:write" || create.Scopes[1] != "blog:admin" {
		t.Errorf("scopes = %v", create.Scopes)
	}
	if create.Example != `{"title":"hi"}` {
		t.Errorf("example = %q", create.Example)
	}
	if len(create.Request) != 2 || create.Request[0].Name != "title" || create.Request[0].Type != "string" {
		t.Errorf("request = %+v", create.Request)
	}
	if len(create.Response) != 1 || create.Response[0].Name != "id" {
		t.Errorf("response = %+v", create.Response)
	}

	read, ok := r.Endpoint("blog.Blog.Read")
	if !ok || read.Description != "Call Blog.Read on blog service" {
		t.Errorf("unexpected read endpoint: %+v", read)
	}

	if !r.HasService("blog") || r.HasService("nope") {
		t.Error("HasService mismatch")
	}
	if got := r.Services(); len(got) != 1 || got[0] != "blog" {
		t.Errorf("Services = %v", got)
	}
	if got := r.EndpointsFor("blog"); len(got) != 2 {
		t.Errorf("EndpointsFor = %d", len(got))
	}
	if s := r.Service("blog"); s == nil || s.Name != "blog" {
		t.Errorf("Service snapshot = %v", s)
	}
}

func TestResolverWatchRefreshesCatalog(t *testing.T) {
	reg := registry.NewMemoryRegistry()
	svc := &registry.Service{
		Name: "blog",
		Nodes: []*registry.Node{{
			Id:      "blog-1",
			Address: "localhost:9090",
		}},
		Endpoints: []*registry.Endpoint{{Name: "Blog.Create"}},
	}
	if err := reg.Register(svc); err != nil {
		t.Fatal(err)
	}

	r := New(reg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r.Start(ctx)

	if _, ok := waitEndpoint(r, "blog.Blog.Create"); !ok {
		t.Fatal("initial refresh did not populate catalog")
	}

	svc2 := &registry.Service{
		Name: "users",
		Nodes: []*registry.Node{{
			Id:      "users-1",
			Address: "localhost:9091",
		}},
		Endpoints: []*registry.Endpoint{{Name: "Users.Get"}},
	}
	if err := reg.Register(svc2); err != nil {
		t.Fatal(err)
	}
	// Memory registry watchers deliver async events; the watch loop retries.
	if _, ok := waitEndpoint(r, "users.Users.Get"); !ok {
		t.Fatal("watch did not pick up new service")
	}

	// De-registering drops the endpoint from the catalog.
	_ = reg.Deregister(svc)
	if waitGone(r, "blog.Blog.Create") {
		t.Fatal("deregistered service still in catalog")
	}
}

func TestJSONType(t *testing.T) {
	cases := map[string]string{
		"string": "string", "int": "integer", "int32": "integer", "uint64": "integer",
		"float32": "number", "float64": "number", "bool": "boolean",
		"map": "object", "unknown": "object",
	}
	for in, want := range cases {
		if got := JSONType(in); got != want {
			t.Errorf("JSONType(%q) = %q, want %q", in, got, want)
		}
	}
}

func waitEndpoint(r *Resolver, name string) (*Endpoint, bool) {
	for i := 0; i < 50; i++ {
		if e, ok := r.Endpoint(name); ok {
			return e, true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil, false
}

func waitGone(r *Resolver, name string) bool {
	for i := 0; i < 50; i++ {
		if _, ok := r.Endpoint(name); !ok {
			return false
		}
		time.Sleep(10 * time.Millisecond)
	}
	return true
}
