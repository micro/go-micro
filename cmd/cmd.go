// Package cmd is an interface for parsing the command line
package cmd

import (
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v5/auth"
	hbroker "go-micro.dev/v5/broker/http"
	nbroker "go-micro.dev/v5/broker/nats"

	"go-micro.dev/v5/broker"
	"go-micro.dev/v5/cache"
	"go-micro.dev/v5/cache/redis"
	"go-micro.dev/v5/client"
	"go-micro.dev/v5/config"
	"go-micro.dev/v5/debug/profile"
	"go-micro.dev/v5/debug/profile/http"
	"go-micro.dev/v5/debug/profile/pprof"
	"go-micro.dev/v5/debug/trace"
	"go-micro.dev/v5/logger"
	"go-micro.dev/v5/registry"
	"go-micro.dev/v5/registry/consul"
	"go-micro.dev/v5/registry/etcd"
	"go-micro.dev/v5/registry/nats"
	"go-micro.dev/v5/selector"
	"go-micro.dev/v5/server"
	"go-micro.dev/v5/store"
	"go-micro.dev/v5/store/mysql"
	"go-micro.dev/v5/transport"
	profileconfig "go-micro.dev/v5/profileconfig"
)

type Cmd interface {
	// The cli app within this cmd
	App() *cli.App
	// Adds options, parses flags and initialize
	// exits on error
	Init(opts ...Option) error
	// Options set within this command
	Options() Options
}

type cmd struct {
	opts Options
	app  *cli.App
}

type Option func(o *Options)

var (
	DefaultCmd = newCmd()

	DefaultFlags = []cli.Flag{
		&cli.StringFlag{
			Name:    "client",
			EnvVars: []string{"MICRO_CLIENT"},
			Usage:   "Client for go-micro; rpc",
		},
		&cli.StringFlag{
			Name:    "client_request_timeout",
			EnvVars: []string{"MICRO_CLIENT_REQUEST_TIMEOUT"},
			Usage:   "Sets the client request timeout. e.g 500ms, 5s, 1m. Default: 5s",
		},
		&cli.IntFlag{
			Name:    "client_retries",
			EnvVars: []string{"MICRO_CLIENT_RETRIES"},
			Value:   client.DefaultRetries,
			Usage:   "Sets the client retries. Default: 1",
		},
		&cli.IntFlag{
			Name:    "client_pool_size",
			EnvVars: []string{"MICRO_CLIENT_POOL_SIZE"},
			Usage:   "Sets the client connection pool size. Default: 1",
		},
		&cli.StringFlag{
			Name:    "client_pool_ttl",
			EnvVars: []string{"MICRO_CLIENT_POOL_TTL"},
			Usage:   "Sets the client connection pool ttl. e.g 500ms, 5s, 1m. Default: 1m",
		},
		&cli.IntFlag{
			Name:    "register_ttl",
			EnvVars: []string{"MICRO_REGISTER_TTL"},
			Value:   60,
			Usage:   "Register TTL in seconds",
		},
		&cli.IntFlag{
			Name:    "register_interval",
			EnvVars: []string{"MICRO_REGISTER_INTERVAL"},
			Value:   30,
			Usage:   "Register interval in seconds",
		},
		&cli.StringFlag{
			Name:    "server",
			EnvVars: []string{"MICRO_SERVER"},
			Usage:   "Server for go-micro; rpc",
		},
		&cli.StringFlag{
			Name:    "server_name",
			EnvVars: []string{"MICRO_SERVER_NAME"},
			Usage:   "Name of the server. go.micro.srv.example",
		},
		&cli.StringFlag{
			Name:    "server_version",
			EnvVars: []string{"MICRO_SERVER_VERSION"},
			Usage:   "Version of the server. 1.1.0",
		},
		&cli.StringFlag{
			Name:    "server_id",
			EnvVars: []string{"MICRO_SERVER_ID"},
			Usage:   "Id of the server. Auto-generated if not specified",
		},
		&cli.StringFlag{
			Name:    "server_address",
			EnvVars: []string{"MICRO_SERVER_ADDRESS"},
			Usage:   "Bind address for the server. 127.0.0.1:8080",
		},
		&cli.StringFlag{
			Name:    "server_advertise",
			EnvVars: []string{"MICRO_SERVER_ADVERTISE"},
			Usage:   "Used instead of the server_address when registering with discovery. 127.0.0.1:8080",
		},
		&cli.StringSliceFlag{
			Name:    "server_metadata",
			EnvVars: []string{"MICRO_SERVER_METADATA"},
			Value:   &cli.StringSlice{},
			Usage:   "A list of key-value pairs defining metadata. version=1.0.0",
		},
		&cli.StringFlag{
			Name:    "broker",
			EnvVars: []string{"MICRO_BROKER"},
			Usage:   "Broker for pub/sub. http, nats, rabbitmq",
		},
		&cli.StringFlag{
			Name:    "broker_address",
			EnvVars: []string{"MICRO_BROKER_ADDRESS"},
			Usage:   "Comma-separated list of broker addresses",
		},
		&cli.StringFlag{
			Name:    "profile",
			Usage:   "Debug profiler for cpu and memory stats",
			EnvVars: []string{"MICRO_DEBUG_PROFILE"},
		},
		&cli.StringFlag{
			Name:    "registry",
			EnvVars: []string{"MICRO_REGISTRY"},
			Usage:   "Registry for discovery. etcd, mdns",
		},
		&cli.StringFlag{
			Name:    "registry_address",
			EnvVars: []string{"MICRO_REGISTRY_ADDRESS"},
			Usage:   "Comma-separated list of registry addresses",
		},
		&cli.StringFlag{
			Name:    "selector",
			EnvVars: []string{"MICRO_SELECTOR"},
			Usage:   "Selector used to pick nodes for querying",
		},
		&cli.StringFlag{
			Name:    "store",
			EnvVars: []string{"MICRO_STORE"},
			Usage:   "Store used for key-value storage",
		},
		&cli.StringFlag{
			Name:    "store_address",
			EnvVars: []string{"MICRO_STORE_ADDRESS"},
			Usage:   "Comma-separated list of store addresses",
		},
		&cli.StringFlag{
			Name:    "store_database",
			EnvVars: []string{"MICRO_STORE_DATABASE"},
			Usage:   "Database option for the underlying store",
		},
		&cli.StringFlag{
			Name:    "store_table",
			EnvVars: []string{"MICRO_STORE_TABLE"},
			Usage:   "Table option for the underlying store",
		},
		&cli.StringFlag{
			Name:    "transport",
			EnvVars: []string{"MICRO_TRANSPORT"},
			Usage:   "Transport mechanism used; http",
		},
		&cli.StringFlag{
			Name:    "transport_address",
			EnvVars: []string{"MICRO_TRANSPORT_ADDRESS"},
			Usage:   "Comma-separated list of transport addresses",
		},
		&cli.StringFlag{
			Name:    "tracer",
			EnvVars: []string{"MICRO_TRACER"},
			Usage:   "Tracer for distributed tracing, e.g. memory, jaeger",
		},
		&cli.StringFlag{
			Name:    "tracer_address",
			EnvVars: []string{"MICRO_TRACER_ADDRESS"},
			Usage:   "Comma-separated list of tracer addresses",
		},
		&cli.StringFlag{
			Name:    "auth",
			EnvVars: []string{"MICRO_AUTH"},
			Usage:   "Auth for role based access control, e.g. service",
		},
		&cli.StringFlag{
			Name:    "auth_id",
			EnvVars: []string{"MICRO_AUTH_ID"},
			Usage:   "Account ID used for client authentication",
		},
		&cli.StringFlag{
			Name:    "auth_secret",
			EnvVars: []string{"MICRO_AUTH_SECRET"},
			Usage:   "Account secret used for client authentication",
		},
		&cli.StringFlag{
			Name:    "auth_namespace",
			EnvVars: []string{"MICRO_AUTH_NAMESPACE"},
			Usage:   "Namespace for the services auth account",
			Value:   "go.micro",
		},
		&cli.StringFlag{
			Name:    "auth_public_key",
			EnvVars: []string{"MICRO_AUTH_PUBLIC_KEY"},
			Usage:   "Public key for JWT auth (base64 encoded PEM)",
		},
		&cli.StringFlag{
			Name:    "auth_private_key",
			EnvVars: []string{"MICRO_AUTH_PRIVATE_KEY"},
			Usage:   "Private key for JWT auth (base64 encoded PEM)",
		},
		&cli.StringFlag{
			Name:    "config",
			EnvVars: []string{"MICRO_CONFIG"},
			Usage:   "The source of the config to be used to get configuration",
		},
	}

	DefaultBrokers = map[string]func(...broker.Option) broker.Broker{
		"memory": broker.NewMemoryBroker,
		"http":   hbroker.NewHttpBroker,
		"nats":   nbroker.NewNatsBroker,
	}

	DefaultClients = map[string]func(...client.Option) client.Client{}

	DefaultRegistries = map[string]func(...registry.Option) registry.Registry{
		"consul": consul.NewConsulRegistry,
		"memory": registry.NewMemoryRegistry,
		"nats":   nats.NewNatsRegistry,
		"mdns":   registry.NewMDNSRegistry,
		"etcd":   etcd.NewEtcdRegistry,
	}

	DefaultSelectors = map[string]func(...selector.Option) selector.Selector{}

	DefaultServers = map[string]func(...server.Option) server.Server{}

	DefaultTransports = map[string]func(...transport.Option) transport.Transport{}

	DefaultStores = map[string]func(...store.Option) store.Store{
		"memory": store.NewMemoryStore,
		"mysql":  mysql.NewMysqlStore,
	}

	DefaultTracers = map[string]func(...trace.Option) trace.Tracer{}

	DefaultAuths = map[string]func(...auth.Option) auth.Auth{}

	DefaultProfiles = map[string]func(...profile.Option) profile.Profile{
		"http":  http.NewProfile,
		"pprof": pprof.NewProfile,
	}

	DefaultConfigs = map[string]func(...config.Option) (config.Config, error){}

	DefaultCaches = map[string]func(...cache.Option) cache.Cache{
		"redis": redis.NewRedisCache,
	}
)

func init() {
	rand.Seed(time.Now().Unix())
}

func newCmd(opts ...Option) Cmd {
	options := Options{
		Auth:      &auth.DefaultAuth,
		Broker:    &broker.DefaultBroker,
		Client:    &client.DefaultClient,
		Registry:  &registry.DefaultRegistry,
		Server:    &server.DefaultServer,
		Selector:  &selector.DefaultSelector,
		Transport: &transport.DefaultTransport,
		Store:     &store.DefaultStore,
		Tracer:    &trace.DefaultTracer,
		Profile:   &profile.DefaultProfile,
		Config:    &config.DefaultConfig,
		Cache:     &cache.DefaultCache,

		Brokers:    DefaultBrokers,
		Clients:    DefaultClients,
		Registries: DefaultRegistries,
		Selectors:  DefaultSelectors,
		Servers:    DefaultServers,
		Transports: DefaultTransports,
		Stores:     DefaultStores,
		Tracers:    DefaultTracers,
		Auths:      DefaultAuths,
		Profiles:   DefaultProfiles,
		Configs:    DefaultConfigs,
		Caches:     DefaultCaches,
	}

	for _, o := range opts {
		o(&options)
	}

	if len(options.Description) == 0 {
		options.Description = "a go-micro service"
	}

	cmd := new(cmd)
	cmd.opts = options
	cmd.app = cli.NewApp()
	cmd.app.Name = cmd.opts.Name
	cmd.app.Version = cmd.opts.Version
	cmd.app.Usage = cmd.opts.Description
	cmd.app.Before = cmd.Before
	cmd.app.Flags = DefaultFlags
	cmd.app.Action = func(c *cli.Context) error {
		return nil
	}

	if len(options.Version) == 0 {
		cmd.app.HideVersion = true
	}

	return cmd
}

func (c *cmd) App() *cli.App {
	return c.app
}

func (c *cmd) Options() Options {
	return c.opts
}

func (c *cmd) Before(ctx *cli.Context) error {
	// If flags are set then use them otherwise do nothing
	var serverOpts []server.Option
	var clientOpts []client.Option

	// Set the client
	if name := ctx.String("client"); len(name) > 0 {
		// only change if we have the client and type differs
		if cl, ok := c.opts.Clients[name]; ok && (*c.opts.Client).String() != name {
			*c.opts.Client = cl()
		}
	}

	// Set the server
	if name := ctx.String("server"); len(name) > 0 {
		// only change if we have the server and type differs
		if s, ok := c.opts.Servers[name]; ok && (*c.opts.Server).String() != name {
			*c.opts.Server = s()
		}
	}

	// Set the store
	if name := ctx.String("store"); len(name) > 0 {
		s, ok := c.opts.Stores[name]
		if !ok {
			return fmt.Errorf("unsupported store: %s", name)
		}

		*c.opts.Store = s(store.WithClient(*c.opts.Client))
	}

	// Set the tracer
	if name := ctx.String("tracer"); len(name) > 0 {
		r, ok := c.opts.Tracers[name]
		if !ok {
			return fmt.Errorf("unsupported tracer: %s", name)
		}

		*c.opts.Tracer = r()
	}

	// Setup auth
	authOpts := []auth.Option{}

	if len(ctx.String("auth_id")) > 0 || len(ctx.String("auth_secret")) > 0 {
		authOpts = append(authOpts, auth.Credentials(
			ctx.String("auth_id"), ctx.String("auth_secret"),
		))
	}
	if len(ctx.String("auth_public_key")) > 0 {
		authOpts = append(authOpts, auth.PublicKey(ctx.String("auth_public_key")))
	}
	if len(ctx.String("auth_private_key")) > 0 {
		authOpts = append(authOpts, auth.PrivateKey(ctx.String("auth_private_key")))
	}
	if len(ctx.String("auth_namespace")) > 0 {
		authOpts = append(authOpts, auth.Namespace(ctx.String("auth_namespace")))
	}
	if name := ctx.String("auth"); len(name) > 0 {
		r, ok := c.opts.Auths[name]
		if !ok {
			return fmt.Errorf("unsupported auth: %s", name)
		}

		*c.opts.Auth = r(authOpts...)
	}

	// Set the registry
	if name := ctx.String("registry"); len(name) > 0 && (*c.opts.Registry).String() != name {
		r, ok := c.opts.Registries[name]
		if !ok {
			return fmt.Errorf("Registry %s not found", name)
		}

		*c.opts.Registry = r()
		serverOpts = append(serverOpts, server.Registry(*c.opts.Registry))
		clientOpts = append(clientOpts, client.Registry(*c.opts.Registry))

		if err := (*c.opts.Selector).Init(selector.Registry(*c.opts.Registry)); err != nil {
			logger.Fatalf("Error configuring registry: %v", err)
		}

		clientOpts = append(clientOpts, client.Selector(*c.opts.Selector))

		if err := (*c.opts.Broker).Init(broker.Registry(*c.opts.Registry)); err != nil {
			logger.Fatalf("Error configuring broker: %v", err)
		}
	}

	// --- Profile Grouping Extension ---
	// Check for new profile flag/env (not just debug profiler)
	profileName := ctx.String("profile")
	if profileName == "" {
		profileName = os.Getenv("MICRO_PROFILE")
	}
	if profileName != "" {
		switch profileName {
		case "local":
			imported := profileconfig.LocalProfile()
			*c.opts.Registry = imported.Registry
			*c.opts.Broker = imported.Broker
			*c.opts.Store = imported.Store
			*c.opts.Transport = imported.Transport
		case "nats":
			imported := profileconfig.NatsProfile()
			*c.opts.Registry = imported.Registry
			*c.opts.Broker = imported.Broker
			*c.opts.Store = imported.Store
			*c.opts.Transport = imported.Transport
		// Add more profiles as needed
		default:
			return fmt.Errorf("unsupported profile: %s", profileName)
		}
	}

	// Set the profile
	if name := ctx.String("profile"); len(name) > 0 {
		p, ok := c.opts.Profiles[name]
		if !ok {
			return fmt.Errorf("unsupported profile: %s", name)
		}

		*c.opts.Profile = p()
	}

	// Set the broker
	if name := ctx.String("broker"); len(name) > 0 && (*c.opts.Broker).String() != name {
		b, ok := c.opts.Brokers[name]
		if !ok {
			return fmt.Errorf("Broker %s not found", name)
		}

		*c.opts.Broker = b()
		serverOpts = append(serverOpts, server.Broker(*c.opts.Broker))
		clientOpts = append(clientOpts, client.Broker(*c.opts.Broker))
	}

	// Set the selector
	if name := ctx.String("selector"); len(name) > 0 && (*c.opts.Selector).String() != name {
		s, ok := c.opts.Selectors[name]
		if !ok {
			return fmt.Errorf("Selector %s not found", name)
		}

		*c.opts.Selector = s(selector.Registry(*c.opts.Registry))

		// No server option here. Should there be?
		clientOpts = append(clientOpts, client.Selector(*c.opts.Selector))
	}

	// Set the transport
	if name := ctx.String("transport"); len(name) > 0 && (*c.opts.Transport).String() != name {
		t, ok := c.opts.Transports[name]
		if !ok {
			return fmt.Errorf("Transport %s not found", name)
		}

		*c.opts.Transport = t()
		serverOpts = append(serverOpts, server.Transport(*c.opts.Transport))
		clientOpts = append(clientOpts, client.Transport(*c.opts.Transport))
	}

	// Parse the server options
	metadata := make(map[string]string)
	for _, d := range ctx.StringSlice("server_metadata") {
		var key, val string
		parts := strings.Split(d, "=")
		key = parts[0]
		if len(parts) > 1 {
			val = strings.Join(parts[1:], "=")
		}
		metadata[key] = val
	}

	if len(metadata) > 0 {
		serverOpts = append(serverOpts, server.Metadata(metadata))
	}

	if len(ctx.String("broker_address")) > 0 {
		if err := (*c.opts.Broker).Init(broker.Addrs(strings.Split(ctx.String("broker_address"), ",")...)); err != nil {
			logger.Fatalf("Error configuring broker: %v", err)
		}
	}

	if len(ctx.String("registry_address")) > 0 {
		if err := (*c.opts.Registry).Init(registry.Addrs(strings.Split(ctx.String("registry_address"), ",")...)); err != nil {
			logger.Fatalf("Error configuring registry: %v", err)
		}
	}

	if len(ctx.String("transport_address")) > 0 {
		if err := (*c.opts.Transport).Init(transport.Addrs(strings.Split(ctx.String("transport_address"), ",")...)); err != nil {
			logger.Fatalf("Error configuring transport: %v", err)
		}
	}

	if len(ctx.String("store_address")) > 0 {
		if err := (*c.opts.Store).Init(store.Nodes(strings.Split(ctx.String("store_address"), ",")...)); err != nil {
			logger.Fatalf("Error configuring store: %v", err)
		}
	}

	if len(ctx.String("store_database")) > 0 {
		if err := (*c.opts.Store).Init(store.Database(ctx.String("store_database"))); err != nil {
			logger.Fatalf("Error configuring store database option: %v", err)
		}
	}

	if len(ctx.String("store_table")) > 0 {
		if err := (*c.opts.Store).Init(store.Table(ctx.String("store_table"))); err != nil {
			logger.Fatalf("Error configuring store table option: %v", err)
		}
	}

	if len(ctx.String("server_name")) > 0 {
		serverOpts = append(serverOpts, server.Name(ctx.String("server_name")))
	}

	if len(ctx.String("server_version")) > 0 {
		serverOpts = append(serverOpts, server.Version(ctx.String("server_version")))
	}

	if len(ctx.String("server_id")) > 0 {
		serverOpts = append(serverOpts, server.Id(ctx.String("server_id")))
	}

	if len(ctx.String("server_address")) > 0 {
		serverOpts = append(serverOpts, server.Address(ctx.String("server_address")))
	}

	if len(ctx.String("server_advertise")) > 0 {
		serverOpts = append(serverOpts, server.Advertise(ctx.String("server_advertise")))
	}

	if ttl := time.Duration(ctx.Int("register_ttl")); ttl >= 0 {
		serverOpts = append(serverOpts, server.RegisterTTL(ttl*time.Second))
	}

	if val := time.Duration(ctx.Int("register_interval")); val >= 0 {
		serverOpts = append(serverOpts, server.RegisterInterval(val*time.Second))
	}

	// client opts
	if r := ctx.Int("client_retries"); r >= 0 {
		clientOpts = append(clientOpts, client.Retries(r))
	}

	if t := ctx.String("client_request_timeout"); len(t) > 0 {
		d, err := time.ParseDuration(t)
		if err != nil {
			return fmt.Errorf("failed to parse client_request_timeout: %v", t)
		}
		clientOpts = append(clientOpts, client.RequestTimeout(d))
	}

	if r := ctx.Int("client_pool_size"); r > 0 {
		clientOpts = append(clientOpts, client.PoolSize(r))
	}

	if t := ctx.String("client_pool_ttl"); len(t) > 0 {
		d, err := time.ParseDuration(t)
		if err != nil {
			return fmt.Errorf("failed to parse client_pool_ttl: %v", t)
		}
		clientOpts = append(clientOpts, client.PoolTTL(d))
	}

	if t := ctx.String("client_pool_close_timeout"); len(t) > 0 {
		d, err := time.ParseDuration(t)
		if err != nil {
			return fmt.Errorf("failed to parse client_pool_close_timeout: %v", t)
		}
		clientOpts = append(clientOpts, client.PoolCloseTimeout(d))
	}

	// We have some command line opts for the server.
	// Lets set it up
	if len(serverOpts) > 0 {
		if err := (*c.opts.Server).Init(serverOpts...); err != nil {
			logger.Fatalf("Error configuring server: %v", err)
		}
	}

	// Use an init option?
	if len(clientOpts) > 0 {
		if err := (*c.opts.Client).Init(clientOpts...); err != nil {
			logger.Fatalf("Error configuring client: %v", err)
		}
	}

	// config
	if name := ctx.String("config"); len(name) > 0 {
		// only change if we have the server and type differs
		if r, ok := c.opts.Configs[name]; ok {
			rc, err := r()
			if err != nil {
				logger.Fatalf("Error configuring config: %v", err)
			}
			*c.opts.Config = rc
		}
	}

	return nil
}

func (c *cmd) Init(opts ...Option) error {
	for _, o := range opts {
		o(&c.opts)
	}
	if len(c.opts.Name) > 0 {
		c.app.Name = c.opts.Name
	}
	if len(c.opts.Version) > 0 {
		c.app.Version = c.opts.Version
	}
	c.app.HideVersion = len(c.opts.Version) == 0
	c.app.Usage = c.opts.Description
	c.app.RunAndExitOnError()
	return nil
}

func DefaultOptions() Options {
	return DefaultCmd.Options()
}

func App() *cli.App {
	return DefaultCmd.App()
}

func Init(opts ...Option) error {
	return DefaultCmd.Init(opts...)
}

func NewCmd(opts ...Option) Cmd {
	return newCmd(opts...)
}

// Register CLI commands
func Register(cmds ...*cli.Command) {
	app := DefaultCmd.App()
	app.Commands = append(app.Commands, cmds...)

	// sort the commands so they're listed in order on the cli
	// todo: move this to micro/cli so it's only run when the
	// commands are printed during "help"
	sort.Slice(app.Commands, func(i, j int) bool {
		return app.Commands[i].Name < app.Commands[j].Name
	})
}
