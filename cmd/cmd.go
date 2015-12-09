package cmd

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"strings"
	"text/tabwriter"
	"text/template"
	"time"

	"github.com/codegangsta/cli"
	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/selector"
	"github.com/micro/go-micro/server"
	"github.com/micro/go-micro/transport"
	"github.com/pborman/uuid"
)

var (
	Flags = []cli.Flag{
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
			Value:  ":0",
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
			Value:  "http",
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
			Value:  "consul",
			Usage:  "Registry for discovery. memory, consul, etcd, kubernetes",
		},
		cli.StringFlag{
			Name:   "registry_address",
			EnvVar: "MICRO_REGISTRY_ADDRESS",
			Usage:  "Comma-separated list of registry addresses",
		},
		cli.StringFlag{
			Name:   "selector",
			EnvVar: "MICRO_SELECTOR",
			Value:  "selector",
			Usage:  "Selector used to pick nodes for querying. random, roundrobin, blacklist",
		},
		cli.StringFlag{
			Name:   "transport",
			EnvVar: "MICRO_TRANSPORT",
			Value:  "http",
			Usage:  "Transport mechanism used; http, rabbitmq, nats",
		},
		cli.StringFlag{
			Name:   "transport_address",
			EnvVar: "MICRO_TRANSPORT_ADDRESS",
			Usage:  "Comma-separated list of transport addresses",
		},

		cli.BoolFlag{
			Name:   "disable_ping",
			EnvVar: "MICRO_DISABLE_PING",
			Usage:  "Disable ping",
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
		"http": broker.NewBroker,
	}

	Registries = map[string]func([]string, ...registry.Option) registry.Registry{
		"consul": registry.NewRegistry,
	}

	Selectors = map[string]func(...selector.Option) selector.Selector{
		"random": selector.NewSelector,
	}

	Transports = map[string]func([]string, ...transport.Option) transport.Transport{
		"http": transport.NewTransport,
	}
)

func init() {
	rand.Seed(time.Now().Unix())
}

// ping informs micro-services about this thing
func ping() {
	type Ping struct {
		Id        string
		Timestamp int64
		Product   string
		Version   string
		Arch      string
		Os        string
	}

	p := Ping{
		Id:      uuid.NewUUID().String(),
		Product: "go-micro",
		Version: "latest",
		Arch:    runtime.GOARCH,
		Os:      runtime.GOOS,
	}

	buf := bytes.NewBuffer(nil)
	cl := &http.Client{}

	fn := func() {
		p.Timestamp = time.Now().Unix()
		b, err := json.Marshal(p)
		if err != nil {
			return
		}
		buf.Reset()
		buf.Write(b)
		rsp, err := cl.Post("https://micro-services.co/_ping", "application/json", buf)
		if err != nil {
			return
		}
		rsp.Body.Close()
	}

	// don't ping unless this thing has lived for 30 seconds
	time.Sleep(time.Second * 30)

	// only ping every 24 hours, be non invasive
	for {
		fn()
		time.Sleep(time.Hour * 24)
	}
}

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

	if s, ok := Selectors[c.String("selector")]; ok {
		selector.DefaultSelector = s(selector.Registry(registry.DefaultRegistry))
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
		server.Version(c.String("server_version")),
		server.Id(c.String("server_id")),
		server.Address(c.String("server_address")),
		server.Advertise(c.String("server_advertise")),
		server.Metadata(metadata),
	)

	client.DefaultClient = client.NewClient()

	if !c.Bool("disable_ping") {
		go ping()
	}

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
