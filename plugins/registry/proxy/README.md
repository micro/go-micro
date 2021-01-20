# Proxy Registry

This is a registry plugin for the micro [proxy](https://micro.mu/docs/proxy.html)

## Usage

Here's a simple usage guide

### Run Proxy

```
# download
go get github.com/micro/micro

# run
micro proxy
```

### Import and Flag plugin

```go
import _ "github.com/asim/go-micro/plugins/registry/proxy"
```

```
go run main.go --registry=proxy
```
