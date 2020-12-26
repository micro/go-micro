# Client

## Contents

- main.go - calls each of the go.micro.srv.example handlers and includes the use of the streaming handler
- codegen - demonstrates how to use code generation to remove boilerplate code
- dc_filter - shows how to use Select filters inside a call wrapper for filtering to the local DC
- dc_selector - is the same as dc_filter but as a Selector implementation itself
- pub - publishes messages using the Publish method. By default encoding in protobuf
- selector - shows how to write and load your own Selector
- wrapper - provides examples for how to use client Wrappers (middleware)

