# noproto

This example demonstrates how to use micro without protobuf.

Use micro with standard go types and use the json codec for marshalling. Services have multiple codecs and use the `Content-Type` 
header to determine which to use. The client sends `Content-Type: application/json`. Because we can marshal standard Go types to 
json there is no code generation or use of protobuf required.

## Contents

- main.go - is a micro greeter service
- client - is a micro json client

## Run the example

Run the service

```shell
go run main.go
```

Run the client

```shell
go run client/main.go
```
