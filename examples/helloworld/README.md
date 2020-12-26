# Hello World

This is hello world using micro

## Contents

- main.go - is the main definition of the service, handler and client
- proto - contains the protobuf definition of the API

## Dependencies

Install the following

- [micro](https://github.com/micro/micro)
- [protoc-gen-micro](https://github.com/micro/protoc-gen-micro)

## Run Service

```shell
micro run service --name helloworld
```

## Query Service

```
micro call helloworld Greeter.Hello '{"name": "John"}'
```
