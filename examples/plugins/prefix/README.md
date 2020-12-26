# Prefix Plugin

The prefix plugin is a micro toolkit plugin which strips a path prefix before continuing with the request

## Usage

Register the plugin before building Micro

```
package main

import (
	"github.com/micro/micro/plugin"
	"github.com/micro/examples/plugins/prefix"
)

func init() {
	plugin.Register(prefix.NewPlugin())
}
```

It can then be applied on the command line like so.

```
micro --path_prefix=/api api
```

### Scoped to API

If you like to only apply the plugin for a specific component you can register it with that specifically. 
For example, below you'll see the plugin registered with the API.

```
package main

import (
	"github.com/micro/micro/api"
	"github.com/micro/examples/plugins/prefix"
)

func init() {
	api.Register(prefix.NewPlugin())
}
```

Here's what the help displays when you do that.

```
$ go run main.go plugin.go api --help
NAME:
   main api - Run the micro API

USAGE:
   main api [command options] [arguments...]

OPTIONS:
   --address 		Set the api address e.g 0.0.0.0:8080 [$MICRO_API_ADDRESS]
   --handler 		Specify the request handler to be used for mapping HTTP requests to services; {api, proxy, rpc} [$MICRO_API_HANDLER]
   --namespace 		Set the namespace used by the API e.g. com.example.api [$MICRO_API_NAMESPACE]
   --cors 		Comma separated whitelist of allowed origins for CORS [$MICRO_API_CORS]
   --path_prefix 	Comma separated list of path prefixes to strip before continuing with request e.g /api,/foo,/bar [$PATH_PREFIX]
```

In this case the usage would be

```
micro api --path_prefix=/api
```
