# Go Micro [![License](https://img.shields.io/:license-apache-blue.svg)](https://opensource.org/licenses/Apache-2.0) [![GoDoc](https://godoc.org/github.com/micro/go-micro?status.svg)](https://godoc.org/github.com/micro/go-micro) [![Travis CI](https://api.travis-ci.org/micro/go-micro.svg?branch=master)](https://travis-ci.org/micro/go-micro) [![Go Report Card](https://goreportcard.com/badge/micro/go-micro)](https://goreportcard.com/report/github.com/micro/go-micro)

Go Micro is a pluggable RPC framework for distributed systems development.

The **micro** philosophy is sane defaults with a pluggable architecture. We provide defaults to get you started quickly but everything can be easily swapped out. It comes with built in support for {json,proto}-rpc encoding, consul or multicast dns for service discovery, http for communication and random hashed client side load balancing.

Plugins are available at [github.com/micro/go-plugins](https://github.com/micro/go-plugins).

Follow us on [Twitter](https://twitter.com/microhq) or join the [Slack](http://slack.micro.mu/) community.

## Features

Go Micro abstracts away the details of distributed systems. Here are the main features.

- **Service Discovery** - Automatic service registration and name resolution
- **Load Balancing** - Client side load balancing built on discovery
- **Message Encoding** - Dynamic encoding based on content-type with protobuf and json support
- **Sync Streaming** - RPC based communication with support for bidirectional streaming
- **Async Messaging** - Native PubSub messaging built in for event driven architectures

## Getting Started

For detailed information on the architecture, installation and use of go-micro checkout the [docs](https://micro.mu/docs).

## Sponsors

Sixt is an Enterprise Sponsor of Micro

<a href="https://micro.mu/blog/2016/04/25/announcing-sixt-sponsorship.html"><img src="https://micro.mu/sixt_logo.png" width=150px height="auto" /></a>

Become a sponsor by backing micro on [Patreon](https://www.patreon.com/microhq)
