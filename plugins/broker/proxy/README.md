# Proxy Broker

This is a broker plugin for the micro [proxy](https://micro.mu/docs/proxy.html)

## Usage

Here's a simple usage guide

### Run Proxy

```
# install micro
go get github.com/micro/micro

# run proxy
micro proxy
```

### Import and Flag plugin

```
import _ "github.com/asim/go-micro/plugins/broker/proxy"
```

```
go run main.go --broker=proxy
```
