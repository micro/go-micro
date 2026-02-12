# Micro [![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

A Go microservices toolkit

## Overview

Micro is a toolkit for Go microservices development. It provides the foundation for building services in the cloud. 
The core of Micro is the [Go Micro](https://github.com/micro/go-micro) framework, which developers import and use in their code to 
write services. Surrounding this we introduce a number of tools to make it easy to serve and consume services. 

## Install the CLI

Install `micro` via `go install`

```
go install go-micro.dev/v5/cmd/micro@v5.16.0
```

> **Note:** Use a specific version instead of `@latest` to avoid module path conflicts. See [releases](https://github.com/micro/go-micro/releases) for the latest version.

Or via install script

```
wget -q  https://raw.githubusercontent.com/micro/micro/master/scripts/install.sh -O - | /bin/bash
```

For releases see the [latest](https://go-micro.dev/releases/latest) tag

## Create a service

Create your service (all setup is now automatic!):

```
micro new helloworld
```

This will:
- Create a new service in the `helloworld` directory
- Automatically run `go mod tidy` and `make proto` for you
- Show the updated project tree including generated files
- Warn you if `protoc` is not installed, with install instructions

## Run the service

Run the service

```
micro run
```

List services to see it's running and registered itself

```
micro services
```

## Describe the service

Describe the service to see available endpoints

```
micro describe helloworld
```

Output

```
{
    "name": "helloworld",
    "version": "latest",
    "metadata": null,
    "endpoints": [
        {
            "request": {
                "name": "Request",
                "type": "Request",
                "values": [
                    {
                        "name": "name",
                        "type": "string",
                        "values": null
                    }
                ]
            },
            "response": {
                "name": "Response",
                "type": "Response",
                "values": [
                    {
                        "name": "msg",
                        "type": "string",
                        "values": null
                    }
                ]
            },
            "metadata": {},
            "name": "Helloworld.Call"
        },
        {
            "request": {
                "name": "Context",
                "type": "Context",
                "values": null
            },
            "response": {
                "name": "Stream",
                "type": "Stream",
                "values": null
            },
            "metadata": {
                "stream": "true"
            },
            "name": "Helloworld.Stream"
        }
    ],
    "nodes": [
        {
            "metadata": {
                "broker": "http",
                "protocol": "mucp",
                "registry": "mdns",
                "server": "mucp",
                "transport": "http"
            },
            "id": "helloworld-31e55be7-ac83-4810-89c8-a6192fb3ae83",
            "address": "127.0.0.1:39963"
        }
    ]
}
```

## Call the service

Call via RPC endpoint

```
micro call helloworld Helloworld.Call '{"name": "Asim"}'
```

## Create a client

Create a client to call the service

```go
package main

import (
        "context"
        "fmt"

        "go-micro.dev/v5"
)

type Request struct {
        Name string
}

type Response struct {
        Message string
}

func main() {
        client := micro.New("helloworld").Client()

        req := client.NewRequest("helloworld", "Helloworld.Call", &Request{Name: "John"})

        var rsp Response

        err := client.Call(context.TODO(), req, &rsp)
        if err != nil {
                fmt.Println(err)
                return
        }

        fmt.Println(rsp.Message)
}
```

## Protobuf 

Use protobuf for code generation with [protoc-gen-micro](https://go-micro.dev/tree/master/cmd/protoc-gen-micro)

## Server

The micro server is an api and web dashboard that provide a fixed entrypoint for seeing and querying services.

Run it like so

```
micro server
```

Then browse to [localhost:8080](http://localhost:8080)

### API Endpoints 

The API provides a fixed HTTP entrypoint for calling services

```
curl http://localhost:8080/api/helloworld/Helloworld/Call -d '{"name": "John"}'
```
See /api for more details and documentation for each service

### Web Dashboard 

The web dashboard provides a modern, secure UI for managing and exploring your Micro services. Major features include:

- **Dynamic Service & Endpoint Forms**: Browse all registered services and endpoints. For each endpoint, a dynamic form is generated for easy testing and exploration.
- **API Documentation**: The `/api` page lists all available services and endpoints, with request/response schemas and a sidebar for quick navigation. A documentation banner explains authentication requirements.
- **JWT Authentication**: All login and token management uses a custom JWT utility. Passwords are securely stored with bcrypt. All `/api/x` endpoints and authenticated pages require an `Authorization: Bearer <token>` header (or `micro_token` cookie as fallback).
- **Token Management**: The `/auth/tokens` page allows you to generate, view (obfuscated), and copy JWT tokens. Tokens are stored and can be revoked. When a user is deleted, all their tokens are revoked immediately.
- **User Management**: The `/auth/users` page allows you to create, list, and delete users. Passwords are never shown or stored in plaintext.
- **Token Revocation**: JWT tokens are stored and checked for revocation on every request. Revoked or deleted tokens are immediately invalidated.
- **Security**: All protected endpoints use consistent authentication logic. Unauthorized or revoked tokens receive a 401 error. All sensitive actions require authentication.
- **Logs & Status**: View service logs and status (PID, uptime, etc) directly from the dashboard.

To get started, run:

```
micro server
```

Then browse to [localhost:8080](http://localhost:8080) and log in with the default admin account (`admin`/`micro`).

> **Note:** See the `/api` page for details on API authentication and how to generate tokens for use with the HTTP API
