# Secure

This example demonstrates how to use tls self signed certs with a micro service. 

The micro transport has a secure option which will generate a cert on startup. Clients will use 
insecure skip verify by default.

## Contents

- srv - greeter server with secure transport that generates a tls self signed cert
- cli - greeter client with secure transport that uses insecure skip verify

## Micro Toolkit

The cli example can be used with the micro toolkit for a secure client

Create a tls.go file

```
package main

import (
	"github.com/asim/go-micro/v3/client"
	"github.com/asim/go-micro/v3/transport"
)

func init() {
	client.DefaultClient.Init(
		client.Transport(
			transport.NewTransport(transport.Secure(true)),
		),
	)
}
```

Build the toolkit with the tls.go file

```
cd github.com/micro/micro
go build -o micro main.go tls.go
```
