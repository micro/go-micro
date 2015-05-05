package cmd

import (
	"os"
	"text/tabwriter"
	"text/template"

	"github.com/codegangsta/cli"
	"github.com/myodc/go-micro/registry"
	"github.com/myodc/go-micro/server"
	"github.com/myodc/go-micro/store"
)

var (
	Flags = []cli.Flag{
		cli.StringFlag{Name: "bind_address", Value: ":0", Usage: "Bind address for the server. 127.0.0.1:8080"},
		cli.StringFlag{Name: "registry", Value: "consul", Usage: "Registry for discovery. kubernetes, consul, etc"},
		cli.StringFlag{Name: "store", Value: "consul", Usage: "Store used as a basic key/value store using consul, memcached, etc"},
	}
)

func Setup(c *cli.Context) error {
	server.Address = c.String("bind_address")

	switch c.String("registry") {
	case "kubernetes":
		registry.DefaultRegistry = registry.NewKubernetesRegistry()
	}

	switch c.String("store") {
	case "memcached":
		store.DefaultStore = store.NewMemcacheStore()
	case "memory":
		store.DefaultStore = store.NewMemoryStore()
	case "etcd":
		store.DefaultStore = store.NewEtcdStore()
	}

	return nil
}

func Init() {
	cli.AppHelpTemplate = `
GLOBAL OPTIONS:
   {{range .Flags}}{{.}}
   {{end}}
`

	cli.HelpPrinter = func(templ string, data interface{}) {
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
