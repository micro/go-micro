// Package plugin provides the ability to load plugins
package plugin

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	pg "plugin"
	"strings"
	"text/template"

	"github.com/micro/go-micro/v2/broker"
	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/client/selector"
	"github.com/micro/go-micro/v2/config/cmd"
	"github.com/micro/go-micro/v2/registry"
	"github.com/micro/go-micro/v2/server"
	"github.com/micro/go-micro/v2/transport"
)

type plugin struct{}

// Init sets up the plugin
func (p *plugin) Init(c *Config) error {
	switch c.Type {
	case "broker":
		pg, ok := c.NewFunc.(func(...broker.Option) broker.Broker)
		if !ok {
			return fmt.Errorf("Invalid plugin %s", c.Name)
		}
		cmd.DefaultBrokers[c.Name] = pg
	case "client":
		pg, ok := c.NewFunc.(func(...client.Option) client.Client)
		if !ok {
			return fmt.Errorf("Invalid plugin %s", c.Name)
		}
		cmd.DefaultClients[c.Name] = pg
	case "registry":
		pg, ok := c.NewFunc.(func(...registry.Option) registry.Registry)
		if !ok {
			return fmt.Errorf("Invalid plugin %s", c.Name)
		}
		cmd.DefaultRegistries[c.Name] = pg

	case "selector":
		pg, ok := c.NewFunc.(func(...selector.Option) selector.Selector)
		if !ok {
			return fmt.Errorf("Invalid plugin %s", c.Name)
		}
		cmd.DefaultSelectors[c.Name] = pg
	case "server":
		pg, ok := c.NewFunc.(func(...server.Option) server.Server)
		if !ok {
			return fmt.Errorf("Invalid plugin %s", c.Name)
		}
		cmd.DefaultServers[c.Name] = pg
	case "transport":
		pg, ok := c.NewFunc.(func(...transport.Option) transport.Transport)
		if !ok {
			return fmt.Errorf("Invalid plugin %s", c.Name)
		}
		cmd.DefaultTransports[c.Name] = pg
	default:
		return fmt.Errorf("Unknown plugin type: %s for %s", c.Type, c.Name)
	}

	return nil
}

// Load loads a plugin created with `go build -buildmode=plugin`
func (p *plugin) Load(path string) (*Config, error) {
	plugin, err := pg.Open(path)
	if err != nil {
		return nil, err
	}
	s, err := plugin.Lookup("Plugin")
	if err != nil {
		return nil, err
	}
	pl, ok := s.(*Config)
	if !ok {
		return nil, errors.New("could not cast Plugin object")
	}
	return pl, nil
}

// Generate creates a go file at the specified path.
// You must use `go build -buildmode=plugin`to build it.
func (p *plugin) Generate(path string, c *Config) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	t, err := template.New(c.Name).Parse(tmpl)
	if err != nil {
		return err
	}
	return t.Execute(f, c)
}

// Build generates a dso plugin using the go command `go build -buildmode=plugin`
func (p *plugin) Build(path string, c *Config) error {
	path = strings.TrimSuffix(path, ".so")

	// create go file in tmp path
	temp := os.TempDir()
	base := filepath.Base(path)
	goFile := filepath.Join(temp, base+".go")

	// generate .go file
	if err := p.Generate(goFile, c); err != nil {
		return err
	}
	// remove .go file
	defer os.Remove(goFile)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil && !os.IsExist(err) {
		return fmt.Errorf("Failed to create dir %s: %v", filepath.Dir(path), err)
	}
	cmd := exec.Command("go", "build", "-buildmode=plugin", "-o", path+".so", goFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
