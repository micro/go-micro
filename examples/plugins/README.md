# Plugins

The micro toolkit supports plugins for the binary itself. These are separate from go-micro plugins.

Plugins can be used to add flags, commands and middleware handlers. An example would be authentication, 
logging, tracing, etc. Existing plugins can be found in [go-plugins/micro](https://github.com/micro/go-plugins/tree/master/micro).

## A simple example

Here's a simple example of a plugin that adds a flag and then prints the value

### The plugin

Create a plugin.go file in the top level dir

```go
package main

import (
	"log"
	"github.com/micro/cli/v2"
	"github.com/micro/micro/v2/plugin"
)

func init() {
	plugin.Register(plugin.NewPlugin(
		plugin.WithName("example"),
		plugin.WithFlag(cli.StringFlag{
			Name:   "example_flag",
			Usage:  "This is an example plugin flag",
			EnvVars: []string{"EXAMPLE_FLAG"},
			Value: "avalue",
		}),
		plugin.WithInit(func(ctx *cli.Context) error {
			log.Println("Got value for example_flag", ctx.String("example_flag"))
			return nil
		}),
	))
}
```

### Build with plugin

Simply build micro with the plugin

```shell
go build -o micro ./main.go ./plugin.go
```

## Go-Micro Plugins

Plugins can be added to go-micro in the following ways. By doing so they'll be available to set via command line args or environment variables.

### Import Plugins

```go
import (
	"github.com/micro/go-micro/v2/config/cmd"
	_ "github.com/micro/go-plugins/broker/rabbitmq"
	_ "github.com/micro/go-plugins/registry/kubernetes"
	_ "github.com/micro/go-plugins/transport/nats"
)

func main() {
	// Parse CLI flags
	cmd.Init()
}
```

The same is achieved when calling ```service.Init```

```go
import (
	"github.com/micro/go-micro/v2"
	_ "github.com/micro/go-plugins/broker/rabbitmq"
	_ "github.com/micro/go-plugins/registry/kubernetes"
	_ "github.com/micro/go-plugins/transport/nats"
)

func main() {
	service := micro.NewService(
		// Set service name
		micro.Name("my.service"),
	)

	// Parse CLI flags
	service.Init()
}
```

### Use via CLI Flags

Activate via a command line flag

```shell
go run service.go --broker=rabbitmq --registry=kubernetes --transport=nats
```

### Use Plugins Directly

CLI Flags provide a simple way to initialise plugins but you can do the same yourself.

```go
import (
	"github.com/micro/go-micro/v2"
	"github.com/micro/go-plugins/registry/kubernetes"
)

func main() {
	registry := kubernetes.NewRegistry() //a default to using env vars for master API

	service := micro.NewService(
		// Set service name
		micro.Name("my.service"),
		// Set service registry
		micro.Registry(registry),
	)
}
```

## Build Pattern

You may want to swap out plugins using automation or add plugins to the micro toolkit. 
An easy way to do this is by maintaining a separate file for plugin imports and including it during the build.

Create file plugins.go
```go
package main

import (
	_ "github.com/micro/go-plugins/broker/rabbitmq"
	_ "github.com/micro/go-plugins/registry/kubernetes"
	_ "github.com/micro/go-plugins/transport/nats"
)
```

Build with plugins.go
```shell
go build -o service main.go plugins.go
```

Run with plugins
```shell
service --broker=rabbitmq --registry=kubernetes --transport=nats
```
