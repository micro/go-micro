// Package plugin provides the ability to load plugins
package plugin

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"strings"
	"text/template"

	"github.com/asim/go-micro/v3/broker"
	"github.com/asim/go-micro/v3/client"
	"github.com/asim/go-micro/v3/cmd"
	"github.com/asim/go-micro/v3/registry"
	"github.com/asim/go-micro/v3/selector"
	"github.com/asim/go-micro/v3/server"
	"github.com/asim/go-micro/v3/transport"
)

// Plugin is a plugin loaded from a file
type Plugin struct {
	// Name of the plugin e.g rabbitmq
	Name string
	// Type of the plugin e.g broker
	Type string
	// Path specifies the import path
	Path string
	// NewFunc creates an instance of the plugin
	NewFunc interface{}
}

// Init sets up the plugin
func Init(p *Plugin) error {
	switch p.Type {
	case "broker":
		pg, ok := p.NewFunc.(func(...broker.Option) broker.Broker)
		if !ok {
			return fmt.Errorf("Invalid plugin %s", p.Name)
		}
		cmd.DefaultBrokers[p.Name] = pg
	case "client":
		pg, ok := p.NewFunc.(func(...client.Option) client.Client)
		if !ok {
			return fmt.Errorf("Invalid plugin %s", p.Name)
		}
		cmd.DefaultClients[p.Name] = pg
	case "registry":
		pg, ok := p.NewFunc.(func(...registry.Option) registry.Registry)
		if !ok {
			return fmt.Errorf("Invalid plugin %s", p.Name)
		}
		cmd.DefaultRegistries[p.Name] = pg

	case "selector":
		pg, ok := p.NewFunc.(func(...selector.Option) selector.Selector)
		if !ok {
			return fmt.Errorf("Invalid plugin %s", p.Name)
		}
		cmd.DefaultSelectors[p.Name] = pg
	case "server":
		pg, ok := p.NewFunc.(func(...server.Option) server.Server)
		if !ok {
			return fmt.Errorf("Invalid plugin %s", p.Name)
		}
		cmd.DefaultServers[p.Name] = pg
	case "transport":
		pg, ok := p.NewFunc.(func(...transport.Option) transport.Transport)
		if !ok {
			return fmt.Errorf("Invalid plugin %s", p.Name)
		}
		cmd.DefaultTransports[p.Name] = pg
	}

	return fmt.Errorf("Unknown plugin type: %s for %s", p.Type, p.Name)
}

// Load loads a plugin created with `go build -buildmode=plugin`
func Load(path string) (*Plugin, error) {
	p, err := plugin.Open(path)
	if err != nil {
		return nil, err
	}
	s, err := p.Lookup("Plugin")
	if err != nil {
		return nil, err
	}
	pl, ok := s.(*Plugin)
	if !ok {
		return nil, errors.New("could not cast Plugin object")
	}
	return pl, nil
}

// Generate creates a go file at the specified path.
// You must use `go build -buildmode=plugin`to build it.
func Generate(path string, p *Plugin) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	t, err := template.New(p.Name).Parse(tmpl)
	if err != nil {
		return err
	}
	return t.Execute(f, p)
}

// Build generates a dso plugin using the go command `go build -buildmode=plugin`
func Build(path string, p *Plugin) error {
	path = strings.TrimSuffix(path, ".so")

	// create go file in tmp path
	temp := os.TempDir()
	base := filepath.Base(path)
	goFile := filepath.Join(temp, base+".go")

	// generate .go file
	if err := Generate(goFile, p); err != nil {
		return err
	}
	// remove .go file
	defer os.Remove(goFile)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil && !os.IsExist(err) {
		return fmt.Errorf("Failed to create dir %s: %v", filepath.Dir(path), err)
	}
	c := exec.Command("go", "build", "-buildmode=plugin", "-o", path+".so", goFile)
	return c.Run()
}
