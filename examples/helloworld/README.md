# Hello World

This is hello world using micro

## Contents

- main.go - is the main definition of the service, handler and client
- proto - contains the protobuf definition of the API

## Dependencies

Install the following

- [micro](https://github.com/asim/go-micro/tree/master/cmd/micro)
- [protoc-gen-micro](https://github.com/asim/go-micro/tree/master/cmd/protoc-gen-micro)

## Run Service

```shell
micro run . --name helloworld
```

## Query Service

```
micro call helloworld Greeter.Hello '{"name": "John"}'
```

## List Services

```shell
micro services
```