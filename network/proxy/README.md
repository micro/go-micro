# Go Proxy [![License](https://img.shields.io/:license-apache-blue.svg)](https://opensource.org/licenses/Apache-2.0) [![GoDoc](https://godoc.org/github.com/micro/go-proxy?status.svg)](https://godoc.org/github.com/micro/go-proxy)

Go Proxy is a proxy library for Go Micro.

## Overview

Go Micro is a distributed systems framework for client/server communication. It handles the details 
around discovery, fault tolerance, rpc communication, etc. We may want to leverage this in broader ecosystems 
which make use of standard http or we may also want to offload a number of requirements to a single proxy.

## Features

- **Transparent Proxy** - Proxy requests to any micro services through a single location. Go Proxy enables 
you to write Go Micro proxies which handle and forward requests. This is good for incorporating wrappers.

- **Single Backend Router** - Enable the single backend router to proxy directly to your local app. The proxy 
allows you to set a router which serves your backend service whether its http, grpc, etc.

- **Protocol Aware Handler** - Set a request handler which speaks your app protocol to make outbound requests. 
Your app may not speak the MUCP protocol so it may be easier to translate internally.

- **Control Planes** - Additionally we support use of control planes to offload many distributed systems concerns.
  * [x] [Consul](https://www.consul.io/docs/connect/native.html) - Using Connect-Native to provide secure mTLS.
  * [x] [NATS](https://nats.io/) - Fully leveraging NATS as the control plane and data plane.

