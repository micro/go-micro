package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseMu(t *testing.T) {
	content := `# Micro configuration
service users
    path ./users
    port 8081

service posts
    path ./posts
    port 8082
    depends users

service web
    path ./web
    port 8089
    depends users posts

env development
    STORE_ADDRESS file://./data
    DEBUG true

env production
    STORE_ADDRESS postgres://localhost/db
`

	tmpDir := t.TempDir()
	muPath := filepath.Join(tmpDir, "micro.mu")
	if err := os.WriteFile(muPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := ParseMu(muPath)
	if err != nil {
		t.Fatalf("ParseMu failed: %v", err)
	}

	// Check services
	if len(cfg.Services) != 3 {
		t.Errorf("expected 3 services, got %d", len(cfg.Services))
	}

	users := cfg.Services["users"]
	if users == nil {
		t.Fatal("users service not found")
	}
	if users.Path != "./users" {
		t.Errorf("users.Path = %q, want %q", users.Path, "./users")
	}
	if users.Port != 8081 {
		t.Errorf("users.Port = %d, want %d", users.Port, 8081)
	}

	posts := cfg.Services["posts"]
	if posts == nil {
		t.Fatal("posts service not found")
	}
	if len(posts.Depends) != 1 || posts.Depends[0] != "users" {
		t.Errorf("posts.Depends = %v, want [users]", posts.Depends)
	}

	web := cfg.Services["web"]
	if web == nil {
		t.Fatal("web service not found")
	}
	if len(web.Depends) != 2 {
		t.Errorf("web.Depends = %v, want [users posts]", web.Depends)
	}

	// Check envs
	if len(cfg.Envs) != 2 {
		t.Errorf("expected 2 envs, got %d", len(cfg.Envs))
	}

	dev := cfg.GetEnv("development")
	if dev == nil {
		t.Fatal("development env not found")
	}
	if dev["STORE_ADDRESS"] != "file://./data" {
		t.Errorf("STORE_ADDRESS = %q, want %q", dev["STORE_ADDRESS"], "file://./data")
	}
	if dev["DEBUG"] != "true" {
		t.Errorf("DEBUG = %q, want %q", dev["DEBUG"], "true")
	}
}

func TestParseJSON(t *testing.T) {
	content := `{
  "services": {
    "users": {
      "path": "./users",
      "port": 8081
    },
    "posts": {
      "path": "./posts",
      "port": 8082,
      "depends": ["users"]
    }
  },
  "env": {
    "development": {
      "STORE_ADDRESS": "file://./data"
    }
  }
}`

	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "micro.json")
	if err := os.WriteFile(jsonPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := ParseJSON(jsonPath)
	if err != nil {
		t.Fatalf("ParseJSON failed: %v", err)
	}

	if len(cfg.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(cfg.Services))
	}

	users := cfg.Services["users"]
	if users == nil {
		t.Fatal("users service not found")
	}
	if users.Port != 8081 {
		t.Errorf("users.Port = %d, want %d", users.Port, 8081)
	}
}

func TestTopologicalSort(t *testing.T) {
	cfg := &Config{
		Services: map[string]*Service{
			"web":   {Name: "web", Depends: []string{"users", "posts"}},
			"posts": {Name: "posts", Depends: []string{"users"}},
			"users": {Name: "users"},
		},
	}

	sorted, err := cfg.TopologicalSort()
	if err != nil {
		t.Fatalf("TopologicalSort failed: %v", err)
	}

	if len(sorted) != 3 {
		t.Fatalf("expected 3 services, got %d", len(sorted))
	}

	// users must come before posts and web
	// posts must come before web
	positions := make(map[string]int)
	for i, svc := range sorted {
		positions[svc.Name] = i
	}

	if positions["users"] > positions["posts"] {
		t.Error("users should come before posts")
	}
	if positions["users"] > positions["web"] {
		t.Error("users should come before web")
	}
	if positions["posts"] > positions["web"] {
		t.Error("posts should come before web")
	}
}

func TestCircularDependency(t *testing.T) {
	cfg := &Config{
		Services: map[string]*Service{
			"a": {Name: "a", Depends: []string{"b"}},
			"b": {Name: "b", Depends: []string{"a"}},
		},
	}

	_, err := cfg.TopologicalSort()
	if err == nil {
		t.Error("expected circular dependency error")
	}
}

func TestLoad(t *testing.T) {
	// Test with no config file
	tmpDir := t.TempDir()
	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg != nil {
		t.Error("expected nil config when no file exists")
	}

	// Test with micro.mu
	muContent := `service test
    path ./test
    port 8080
`
	if err := os.WriteFile(filepath.Join(tmpDir, "micro.mu"), []byte(muContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err = Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected config to be loaded")
	}
	if cfg.Services["test"] == nil {
		t.Error("test service not found")
	}
}
