// Package config handles micro.mu and micro.json configuration parsing
package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config represents the micro run configuration
type Config struct {
	Services map[string]*Service         `json:"services"`
	Envs     map[string]map[string]string `json:"env"`
	Deploy   map[string]*DeployTarget     `json:"deploy"`
}

// DeployTarget represents a deployment target configuration
type DeployTarget struct {
	Name string `json:"-"`
	SSH  string `json:"ssh"`
	Path string `json:"path,omitempty"`
}

// Service represents a service configuration
type Service struct {
	Name    string   `json:"-"`
	Path    string   `json:"path"`
	Port    int      `json:"port,omitempty"`
	Depends []string `json:"depends,omitempty"`
}

// Load attempts to load configuration from micro.mu or micro.json in the given directory
func Load(dir string) (*Config, error) {
	// Try micro.mu first (preferred)
	muPath := filepath.Join(dir, "micro.mu")
	if _, err := os.Stat(muPath); err == nil {
		return ParseMu(muPath)
	}

	// Fall back to micro.json
	jsonPath := filepath.Join(dir, "micro.json")
	if _, err := os.Stat(jsonPath); err == nil {
		return ParseJSON(jsonPath)
	}

	return nil, nil // No config file, not an error
}

// ParseJSON parses a micro.json configuration file
func ParseJSON(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}

	// Set service names from map keys
	for name, svc := range cfg.Services {
		svc.Name = name
	}

	return &cfg, nil
}

// ParseMu parses a micro.mu DSL configuration file
//
// Format:
//
//	service users
//	    path ./users
//	    port 8081
//
//	service posts
//	    path ./posts
//	    port 8082
//	    depends users
//
//	env development
//	    STORE_ADDRESS file://./data
func ParseMu(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer file.Close()

	cfg := &Config{
		Services: make(map[string]*Service),
		Envs:     make(map[string]map[string]string),
		Deploy:   make(map[string]*DeployTarget),
	}

	var currentService *Service
	var currentEnv string
	var currentEnvMap map[string]string
	var currentDeploy *DeployTarget

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip empty lines and comments
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Check indentation
		indented := strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t")

		if !indented {
			// Top-level declaration
			parts := strings.Fields(trimmed)
			if len(parts) < 2 {
				return nil, fmt.Errorf("%s:%d: expected 'service <name>' or 'env <name>'", path, lineNum)
			}

			keyword := parts[0]
			name := parts[1]

			switch keyword {
			case "service":
				// Save previous env if any
				if currentEnv != "" && currentEnvMap != nil {
					cfg.Envs[currentEnv] = currentEnvMap
				}
				currentEnv = ""
				currentEnvMap = nil

				currentService = &Service{Name: name}
				cfg.Services[name] = currentService

			case "env":
				// Save previous env if any
				if currentEnv != "" && currentEnvMap != nil {
					cfg.Envs[currentEnv] = currentEnvMap
				}
				currentService = nil
				currentDeploy = nil
				currentEnv = name
				currentEnvMap = make(map[string]string)

			case "deploy":
				// Save previous env if any
				if currentEnv != "" && currentEnvMap != nil {
					cfg.Envs[currentEnv] = currentEnvMap
				}
				currentService = nil
				currentEnv = ""
				currentEnvMap = nil
				currentDeploy = &DeployTarget{Name: name}
				cfg.Deploy[name] = currentDeploy

			default:
				return nil, fmt.Errorf("%s:%d: unknown keyword '%s'", path, lineNum, keyword)
			}
		} else {
			// Indented property
			parts := strings.Fields(trimmed)
			if len(parts) < 2 {
				return nil, fmt.Errorf("%s:%d: expected 'key value'", path, lineNum)
			}

			key := parts[0]
			value := strings.Join(parts[1:], " ")

			if currentService != nil {
				switch key {
				case "path":
					currentService.Path = value
				case "port":
					port, err := strconv.Atoi(value)
					if err != nil {
						return nil, fmt.Errorf("%s:%d: invalid port '%s'", path, lineNum, value)
					}
					currentService.Port = port
				case "depends":
					currentService.Depends = parts[1:]
				default:
					return nil, fmt.Errorf("%s:%d: unknown service property '%s'", path, lineNum, key)
				}
			} else if currentDeploy != nil {
				switch key {
				case "ssh":
					currentDeploy.SSH = value
				case "path":
					currentDeploy.Path = value
				default:
					return nil, fmt.Errorf("%s:%d: unknown deploy property '%s'", path, lineNum, key)
				}
			} else if currentEnvMap != nil {
				// Environment variable
				currentEnvMap[key] = value
			} else {
				return nil, fmt.Errorf("%s:%d: property outside of service, deploy, or env block", path, lineNum)
			}
		}
	}

	// Save final env if any
	if currentEnv != "" && currentEnvMap != nil {
		cfg.Envs[currentEnv] = currentEnvMap
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s: %w", path, err)
	}

	return cfg, nil
}

// TopologicalSort returns services in dependency order
func (c *Config) TopologicalSort() ([]*Service, error) {
	if c == nil || len(c.Services) == 0 {
		return nil, nil
	}

	// Build adjacency list and in-degree count
	inDegree := make(map[string]int)
	for name := range c.Services {
		inDegree[name] = 0
	}

	for _, svc := range c.Services {
		for _, dep := range svc.Depends {
			if _, ok := c.Services[dep]; !ok {
				return nil, fmt.Errorf("service '%s' depends on unknown service '%s'", svc.Name, dep)
			}
			inDegree[svc.Name]++
		}
	}

	// Kahn's algorithm
	var queue []string
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	var result []*Service
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		result = append(result, c.Services[name])

		// Reduce in-degree for dependents
		for _, svc := range c.Services {
			for _, dep := range svc.Depends {
				if dep == name {
					inDegree[svc.Name]--
					if inDegree[svc.Name] == 0 {
						queue = append(queue, svc.Name)
					}
				}
			}
		}
	}

	if len(result) != len(c.Services) {
		return nil, fmt.Errorf("circular dependency detected")
	}

	return result, nil
}

// GetEnv returns environment variables for the given environment name
func (c *Config) GetEnv(name string) map[string]string {
	if c == nil || c.Envs == nil {
		return nil
	}
	return c.Envs[name]
}
