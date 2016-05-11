# Examples

This directory contains example usage of Micro

**Client** contains examples usage of the Client package to call a service.
  - main.go - calls each of the go.micro.srv.example handlers and includes the use of the streaming handler
  - codegen - demonstrates how to use code generation to remove boilerplate code
  - dc_filter - shows how to use Select filters inside a call wrapper for filtering to the local DC
  - dc_selector - is the same as dc_filter but as a Selector implementation itself
  - pub - publishes messages using the Publish method. By default encoding in protobuf
  - selector - shows how to write and load your own Selector
  - wrapper - provides examples for how to use client Wrappers (middleware)

**PubSub** contains an example of using the Broker for Publish and Subscribing.
  - main.go - demonstrates simple runs pub-sub as two go routines running for 10 seconds.
  - producer - publishes messages to the broker every second
  - consumer - consumes any messages sent by the producer

**Server** contains example usage of the Server package to server requests.
  - main.go - initialises and runs the the server
  - handler - is an example RPC request handler for the Server
  - proto - contains the protobuf defintion for the Server API
  - subscriber - is a handler for subscribing via the Server
  - wrapper - demonstrates use of a server HandlerWrapper
  - codegen - shows how to use codegenerated registration to reduce boilerplate

**Service** contains example usage of the top level Service in go-micro.
  - main.go - is the main definition of the service, handler and client
  - proto - contains the protobuf definition of the API
  - wrapper - demonstrates the use of Client and Server Wrappers
