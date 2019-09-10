// Package plugin provides the ability to load plugins
package plugin

// Plugin is a plugin loaded from a file
type Plugin interface {
	// Initialise a plugin with the config
	Init(c *Config) error
	// Load loads a .so plugin at the given path
	Load(path string) (*Config, error)
	// Build a .so plugin with config at the path specified
	Build(path string, c *Config) error
}

// Config is the plugin config
type Config struct {
	// Name of the plugin e.g rabbitmq
	Name string
	// Type of the plugin e.g broker
	Type string
	// Path specifies the import path
	Path string
	// NewFunc creates an instance of the plugin
	NewFunc interface{}
}

var (
	// Default plugin loader
	DefaultPlugin = NewPlugin()
)

// NewPlugin creates a new plugin interface
func NewPlugin() Plugin {
	return &plugin{}
}

func Build(path string, c *Config) error {
	return DefaultPlugin.Build(path, c)
}

func Load(path string) (*Config, error) {
	return DefaultPlugin.Load(path)
}

func Init(c *Config) error {
	return DefaultPlugin.Init(c)
}
