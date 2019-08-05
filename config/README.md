# Config [![GoDoc](https://godoc.org/github.com/micro/go-micro/config?status.svg)](https://godoc.org/github.com/micro/go-micro/config)

Config is a pluggable dynamic config package

Most config in applications are statically configured or include complex logic to load from multiple sources. 
Go Config makes this easy, pluggable and mergeable. You'll never have to deal with config in the same way again.

## Features

- **Dynamic Loading** - Load configuration from multiple source as and when needed. Go Config manages watching config sources 
in the background and automatically merges and updates an in memory view. 

- **Pluggable Sources** - Choose from any number of sources to load and merge config. The backend source is abstracted away into 
a standard format consumed internally and decoded via encoders. Sources can be env vars, flags, file, etcd, k8s configmap, etc.

- **Mergeable Config** - If you specify multiple sources of config, regardless of format, they will be merged and presented in 
a single view. This massively simplifies priority order loading and changes based on environment.

- **Observe Changes** - Optionally watch the config for changes to specific values. Hot reload your app using Go Config's watcher. 
You don't have to handle ad-hoc hup reloading or whatever else, just keep reading the config and watch for changes if you need 
to be notified.

- **Sane Defaults** - In case config loads badly or is completely wiped away for some unknown reason, you can specify fallback 
values when accessing any config values directly. This ensures you'll always be reading some sane default in the event of a problem.

## Getting Started

For detailed information or architecture, installation and general usage see the [docs](https://micro.mu/docs/go-config.html)

