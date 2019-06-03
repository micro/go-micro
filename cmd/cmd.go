// Package cmd is an interface for parsing the command line
package cmd

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/micro/cli"
	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/server"
	"github.com/micro/go-micro/util/log"

	// brokers
	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/broker/http"
	"github.com/micro/go-micro/broker/memory"
	"github.com/micro/go-micro/broker/nats"

	// registries
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/registry/consul"
	"github.com/micro/go-micro/registry/gossip"
	"github.com/micro/go-micro/registry/mdns"
	rmem "github.com/micro/go-micro/registry/memory"

	// selectors
	"github.com/micro/go-micro/selector"
	"github.com/micro/go-micro/selector/dns"
	"github.com/micro/go-micro/selector/static"

	// transports
	"github.com/micro/go-micro/transport"
	tgrpc "github.com/micro/go-micro/transport/grpc"
	thttp "github.com/micro/go-micro/transport/http"
	tmem "github.com/micro/go-micro/transport/memory"
)

type Cmd interface {
	// The cli app within this cmd
	App() *cli.App
	// Adds options, parses flags and initialise
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
		cli.StringFlag{
			Name:   "client",
			EnvVar: "MICRO_CLIENT",
			Usage:  "Client for go-micro; rpc",
		},
		cli.StringFlag{
			Name:   "client_request_timeout",
			EnvVar: "MICRO_CLIENT_REQUEST_TIMEOUT",
			Usage:  "Sets the client request timeout. e.g 500ms, 5s, 1m. Default: 5s",
		},
		cli.IntFlag{
			Name:   "client_retries",
			EnvVar: "MICRO_CLIENT_RETRIES",
			Value:  client.DefaultRetries,
			Usage:  "Sets the client retries. Default: 1",
		},
		cli.IntFlag{
			Name:   "client_pool_size",
			EnvVar: "MICRO_CLIENT_POOL_SIZE",
			Usage:  "Sets the client connection pool size. Default: 1",
		},
		cli.StringFlag{
			Name:   "client_pool_ttl",
			EnvVar: "MICRO_CLIENT_POOL_TTL",
			Usage:  "Sets the client connection pool ttl. e.g 500ms, 5s, 1m. Default: 1m",
		},
		cli.IntFlag{
			Name:   "register_ttl",
			EnvVar: "MICRO_REGISTER_TTL",
			Usage:  "Register TTL in seconds",
		},
		cli.IntFlag{
			Name:   "register_interval",
			EnvVar: "MICRO_REGISTER_INTERVAL",
			Usage:  "Register interval in seconds",
		},
		cli.StringFlag{
			Name:   "server",
			EnvVar: "MICRO_SERVER",
			Usage:  "Server for go-micro; rpc",
		},
		cli.StringFlag{
			Name:   "server_name",
			EnvVar: "MICRO_SERVER_NAME",
			Usage:  "Name of the server. go.micro.srv.example",
		},
		cli.StringFlag{
			Name:   "server_version",
			EnvVar: "MICRO_SERVER_VERSION",
			Usage:  "Version of the server. 1.1.0",
		},
		cli.StringFlag{
			Name:   "server_id",
			EnvVar: "MICRO_SERVER_ID",
			Usage:  "Id of the server. Auto-generated if not specified",
		},
		cli.StringFlag{
			Name:   "server_address",
			EnvVar: "MICRO_SERVER_ADDRESS",
			Usage:  "Bind address for the server. 127.0.0.1:8080",
		},
		cli.StringFlag{
			Name:   "server_advertise",
			EnvVar: "MICRO_SERVER_ADVERTISE",
			Usage:  "Used instead of the server_address when registering with discovery. 127.0.0.1:8080",
		},
		cli.StringSliceFlag{
			Name:   "server_metadata",
			EnvVar: "MICRO_SERVER_METADATA",
			Value:  &cli.StringSlice{},
			Usage:  "A list of key-value pairs defining metadata. version=1.0.0",
		},
		cli.StringFlag{
			Name:   "broker",
			EnvVar: "MICRO_BROKER",
			Usage:  "Broker for pub/sub. http, nats, rabbitmq",
		},
		cli.StringFlag{
			Name:   "broker_address",
			EnvVar: "MICRO_BROKER_ADDRESS",
			Usage:  "Comma-separated list of broker addresses",
		},
		cli.StringFlag{
			Name:   "registry",
			EnvVar: "MICRO_REGISTRY",
			Usage:  "Registry for discovery. consul, mdns",
		},
		cli.StringFlag{
			Name:   "registry_address",
			EnvVar: "MICRO_REGISTRY_ADDRESS",
			Usage:  "Comma-separated list of registry addresses",
		},
		cli.StringFlag{
			Name:   "selector",
			EnvVar: "MICRO_SELECTOR",
			Usage:  "Selector used to pick nodes for querying",
		},
		cli.StringFlag{
			Name:   "transport",
			EnvVar: "MICRO_TRANSPORT",
			Usage:  "Transport mechanism used; http",
		},
		cli.StringFlag{
			Name:   "transport_address",
			EnvVar: "MICRO_TRANSPORT_ADDRESS",
			Usage:  "Comma-separated list of transport addresses",
		},
	}

	DefaultBrokers = map[string]func(...broker.Option) broker.Broker{
		"http":   http.NewBroker,
		"memory": memory.NewBroker,
		"nats":   nats.NewBroker,
	}

	DefaultClients = map[string]func(...client.Option) client.Client{
		"rpc": client.NewClient,
	}

	DefaultRegistries = map[string]func(...registry.Option) registry.Registry{
		"consul": consul.NewRegistry,
		"gossip": gossip.NewRegistry,
		"mdns":   mdns.NewRegistry,
		"memory": rmem.NewRegistry,
	}

	DefaultSelectors = map[string]func(...selector.Option) selector.Selector{
		"default": selector.NewSelector,
		"dns":     dns.NewSelector,
		"cache":   selector.NewSelector,
		"static":  static.NewSelector,
	}

	DefaultServers = map[string]func(...server.Option) server.Server{
		"rpc": server.NewServer,
	}

	DefaultTransports = map[string]func(...transport.Option) transport.Transport{
		"memory": tmem.NewTransport,
		"http":   thttp.NewTransport,
		"grpc":   tgrpc.NewTransport,
	}

	// used for default selection as the fall back
	defaultClient    = "rpc"
	defaultServer    = "rpc"
	defaultBroker    = "http"
	defaultRegistry  = "mdns"
	defaultSelector  = "registry"
	defaultTransport = "http"
)

func init() {
	rand.Seed(time.Now().Unix())
	help := cli.HelpPrinter
	cli.HelpPrinter = func(writer io.Writer, templ string, data interface{}) {
		help(writer, templ, data)
		os.Exit(0)
	}
}

func newCmd(opts ...Option) Cmd {
	options := Options{
		Broker:    &broker.DefaultBroker,
		Client:    &client.DefaultClient,
		Registry:  &registry.DefaultRegistry,
		Server:    &server.DefaultServer,
		Selector:  &selector.DefaultSelector,
		Transport: &transport.DefaultTransport,

		Brokers:    DefaultBrokers,
		Clients:    DefaultClients,
		Registries: DefaultRegistries,
		Selectors:  DefaultSelectors,
		Servers:    DefaultServers,
		Transports: DefaultTransports,
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
	cmd.app.Action = func(c *cli.Context) {}

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
			log.Fatalf("Error configuring registry: %v", err)
		}

		clientOpts = append(clientOpts, client.Selector(*c.opts.Selector))

		if err := (*c.opts.Broker).Init(broker.Registry(*c.opts.Registry)); err != nil {
			log.Fatalf("Error configuring broker: %v", err)
		}
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
			log.Fatalf("Error configuring broker: %v", err)
		}
	}

	if len(ctx.String("registry_address")) > 0 {
		if err := (*c.opts.Registry).Init(registry.Addrs(strings.Split(ctx.String("registry_address"), ",")...)); err != nil {
			log.Fatalf("Error configuring registry: %v", err)
		}
	}

	if len(ctx.String("transport_address")) > 0 {
		if err := (*c.opts.Transport).Init(transport.Addrs(strings.Split(ctx.String("transport_address"), ",")...)); err != nil {
			log.Fatalf("Error configuring transport: %v", err)
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

	if ttl := time.Duration(ctx.GlobalInt("register_ttl")); ttl > 0 {
		serverOpts = append(serverOpts, server.RegisterTTL(ttl*time.Second))
	}

	if val := time.Duration(ctx.GlobalInt("register_interval")); val > 0 {
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

	// We have some command line opts for the server.
	// Lets set it up
	if len(serverOpts) > 0 {
		if err := (*c.opts.Server).Init(serverOpts...); err != nil {
			log.Fatalf("Error configuring server: %v", err)
		}
	}

	// Use an init option?
	if len(clientOpts) > 0 {
		if err := (*c.opts.Client).Init(clientOpts...); err != nil {
			log.Fatalf("Error configuring client: %v", err)
		}
	}

	return nil
}

func (c *cmd) Init(opts ...Option) error {
	for _, o := range opts {
		o(&c.opts)
	}
	c.app.Name = c.opts.Name
	c.app.Version = c.opts.Version
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
