package cmd

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"text/template"

	"github.com/codegangsta/cli"
	"github.com/myodc/go-micro/broker"
	"github.com/myodc/go-micro/client"
	"github.com/myodc/go-micro/registry"
	"github.com/myodc/go-micro/server"
	"github.com/myodc/go-micro/transport"

	// brokers
	"github.com/myodc/go-micro/broker/http"
	"github.com/myodc/go-micro/broker/nats"
	"github.com/myodc/go-micro/broker/rabbitmq"

	// registries
	"github.com/myodc/go-micro/registry/consul"
	"github.com/myodc/go-micro/registry/etcd"
	"github.com/myodc/go-micro/registry/memory"

	// transport
	thttp "github.com/myodc/go-micro/transport/http"
	tnats "github.com/myodc/go-micro/transport/nats"
	trmq "github.com/myodc/go-micro/transport/rabbitmq"
)

var (
	Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "server_name",
			EnvVar: "MICRO_SERVER_NAME",
			Usage:  "Name of the server. go.micro.srv.example",
		},
		cli.StringFlag{
			Name:   "server_id",
			EnvVar: "MICRO_SERVER_ID",
			Usage:  "Id of the server. Auto-generated if not specified",
		},
		cli.StringFlag{
			Name:   "server_address",
			EnvVar: "MICRO_SERVER_ADDRESS",
			Value:  ":0",
			Usage:  "Bind address for the server. 127.0.0.1:8080",
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
			Value:  "http",
			Usage:  "Broker for pub/sub. http, nats, etc",
		},
		cli.StringFlag{
			Name:   "broker_address",
			EnvVar: "MICRO_BROKER_ADDRESS",
			Usage:  "Comma-separated list of broker addresses",
		},
		cli.StringFlag{
			Name:   "registry",
			EnvVar: "MICRO_REGISTRY",
			Value:  "consul",
			Usage:  "Registry for discovery. memory, consul, etcd",
		},
		cli.StringFlag{
			Name:   "registry_address",
			EnvVar: "MICRO_REGISTRY_ADDRESS",
			Usage:  "Comma-separated list of registry addresses",
		},
		cli.StringFlag{
			Name:   "transport",
			EnvVar: "MICRO_TRANSPORT",
			Value:  "http",
			Usage:  "Transport mechanism used; http, rabbitmq, etc",
		},
		cli.StringFlag{
			Name:   "transport_address",
			EnvVar: "MICRO_TRANSPORT_ADDRESS",
			Usage:  "Comma-separated list of transport addresses",
		},

		// logging flags
		cli.BoolFlag{
			Name:  "logtostderr",
			Usage: "log to standard error instead of files",
		},
		cli.BoolFlag{
			Name:  "alsologtostderr",
			Usage: "log to standard error as well as files",
		},
		cli.StringFlag{
			Name:  "log_dir",
			Usage: "log files will be written to this directory instead of the default temporary directory",
		},
		cli.StringFlag{
			Name:  "stderrthreshold",
			Usage: "logs at or above this threshold go to stderr",
		},
		cli.StringFlag{
			Name:  "v",
			Usage: "log level for V logs",
		},
		cli.StringFlag{
			Name:  "vmodule",
			Usage: "comma-separated list of pattern=N settings for file-filtered logging",
		},
		cli.StringFlag{
			Name:  "log_backtrace_at",
			Usage: "when logging hits line file:N, emit a stack trace",
		},
	}

	Brokers = map[string]func([]string, ...broker.Option) broker.Broker{
		"http":     http.NewBroker,
		"nats":     nats.NewBroker,
		"rabbitmq": rabbitmq.NewBroker,
	}

	Registries = map[string]func([]string, ...registry.Option) registry.Registry{
		"consul": consul.NewRegistry,
		"etcd":   etcd.NewRegistry,
		"memory": memory.NewRegistry,
	}

	Transports = map[string]func([]string, ...transport.Option) transport.Transport{
		"http":     thttp.NewTransport,
		"rabbitmq": trmq.NewTransport,
		"nats":     tnats.NewTransport,
	}
)

func Setup(c *cli.Context) error {
	os.Args = os.Args[:1]

	flag.Set("logtostderr", fmt.Sprintf("%v", c.Bool("logtostderr")))
	flag.Set("alsologtostderr", fmt.Sprintf("%v", c.Bool("alsologtostderr")))
	flag.Set("stderrthreshold", c.String("stderrthreshold"))
	flag.Set("log_backtrace_at", c.String("log_backtrace_at"))
	flag.Set("log_dir", c.String("log_dir"))
	flag.Set("vmodule", c.String("vmodule"))
	flag.Set("v", c.String("v"))

	flag.Parse()

	if b, ok := Brokers[c.String("broker")]; ok {
		broker.DefaultBroker = b(strings.Split(c.String("broker_address"), ","))
	}

	if r, ok := Registries[c.String("registry")]; ok {
		registry.DefaultRegistry = r(strings.Split(c.String("registry_address"), ","))
	}

	if t, ok := Transports[c.String("transport")]; ok {
		transport.DefaultTransport = t(strings.Split(c.String("transport_address"), ","))
	}

	metadata := make(map[string]string)
	for _, d := range c.StringSlice("server_metadata") {
		var key, val string
		parts := strings.Split(d, "=")
		key = parts[0]
		if len(parts) > 1 {
			val = strings.Join(parts[1:], "=")
		}
		metadata[key] = val
	}

	server.DefaultServer = server.NewServer(
		server.Name(c.String("server_name")),
		server.Id(c.String("server_id")),
		server.Address(c.String("server_address")),
		server.Metadata(metadata),
	)

	client.DefaultClient = client.NewClient()

	return nil
}

func Init() {
	cli.AppHelpTemplate = `
GLOBAL OPTIONS:
   {{range .Flags}}{{.}}
   {{end}}
`

	cli.HelpPrinter = func(writer io.Writer, templ string, data interface{}) {
		w := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', 0)
		t := template.Must(template.New("help").Parse(templ))
		err := t.Execute(w, data)
		if err != nil {
			panic(err)
		}
		w.Flush()
		os.Exit(2)
	}

	app := cli.NewApp()
	app.HideVersion = true
	app.Usage = "a go micro app"
	app.Action = func(c *cli.Context) {}
	app.Before = Setup
	app.Flags = Flags
	app.RunAndExitOnError()
}
