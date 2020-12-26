# Template Function

This is the Template function

Generated with

```
micro new github.com/micro/go-micro/examples/template/fnc --namespace=go.micro --alias=template --type=fnc --plugin=registry=etcd
```

## Getting Started

- [Configuration](#configuration)
- [Dependencies](#dependencies)
- [Usage](#usage)

## Configuration

- FQDN: go.micro.fnc.template
- Type: fnc
- Alias: template

## Usage

A Makefile is included for convenience

Build the binary

```
make build
```

Run the function once
```
./template-fnc
```

Build a docker image
```
make docker
```
