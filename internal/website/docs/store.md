# Store

The store provides a pluggable interface for data storage in Go Micro.

## Features
- Key-value storage
- Multiple backend support

## Implementations
Supported stores include:
- Memory (default)
- File
- MySQL
- Redis

Configure the store as needed for your application.

## Example Usage

Here's how to use the store in your Go Micro service:

```go
package main

import (
    "go-micro.dev/v5"
    "go-micro.dev/v5/store"
    "log"
)

func main() {
    service := micro.NewService()
    service.Init()

    // Write a record
    if err := store.Write(&store.Record{Key: "foo", Value: []byte("bar")}); err != nil {
        log.Fatal(err)
    }

    // Read a record
    recs, err := store.Read("foo")
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Read value: %s", string(recs[0].Value))
}
```
