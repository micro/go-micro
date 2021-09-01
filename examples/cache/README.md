# Cache

This is an example of a cache service using the [asim/go-micro/cache][1] package.

## Contents

This project was generated using [Gomu][2].

* handler - contains the service handler
* proto - contains the protocol buffer and generated code

## Usage

Run the `go.micro.srv.cache` service using [Gomu][2].

```bash
gomu run
```

You can also run it using plain Go.

```bash
go run main.go
```

We'll be using [Gomu][2] to call the service. You can store a new key-value
pair in the cache.

```bash
gomu call go.micro.srv.cache Cache.Put '{"key":"test","value":"hello go-micro","duration":"12h"}'
```

You can get values from the cache.

```bash
$ gomu call go.micro.srv.cache Cache.Get '{"key":"test"}'
{"expiration":"2021-09-01 22:42:24.2370591 +0200 CEST","value":"hello go-micro"}
```

Finally you can delete keys from the cache.

```bash
gomu call go.micro.srv.cache Cache.Delete '{"key":"test"}'
```

[1]: https://github.com/asim/go-micro/tree/master/cache
[2]: https://github.com/asim/go-micro/tree/master/cmd/gomu
