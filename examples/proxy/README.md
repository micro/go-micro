# Proxy Sidecar

The [micro proxy](https://github.com/micro/micro/tree/master/proxy) provides [go-micro](https://github.com/asim/go-micro) features as http endpoints.

This directory contains examples for using the proxy with various languages.

## Usage

See details below

### Run Proxy

```
micro proxy
```

Or with http proxy handler
```
micro proxy --handler=http
```

### Service

Run server
```
{python, ruby} {http, rpc}_server.{py, rb}
```

Run client
```
{python, ruby} {http, rpc}_client.{py, rb}
```

## Examples

Each language directory {python, ruby, ...} contains examples for the following:

- Registering Service
- JSON RPC Server and Client
- HTTP Server and Client
