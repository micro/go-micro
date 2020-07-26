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
)

type plugin struct{}

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
