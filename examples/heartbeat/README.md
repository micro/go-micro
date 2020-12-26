# Heartbeat

A demonstration of using heartbeating with service discovery.

## Rationale

Services register with service discovery on startup and deregister on shutdown. Sometimes these services may unexpectedly die or 
be killed forcefully or face transient network issues. In these cases stale nodes will be left in service discovery. It would be 
ideal if services were automatically removed.

## Solution

Micro supports the option of a register TTL and register interval for this exact reason. TTL specifies how long a registration should 
exist in discovery after which it expires and is removed. Interval is the time at which a service should re-register to preserve 
it's registration in service discovery.

These are options made available in go-micro and as flags in the micro toolkit

## Toolkit

Run any component of the toolkit with the flags like so

```
micro --register_ttl=30 --register_interval=15 api
```

This example shows that we're setting a ttl of 30 seconds with a re-register interval of 15 seconds.

## Go Micro

When declaring a micro service you can pass in the options as time.Duration

```
service := micro.NewService(
	micro.Name("com.example.srv.foo"),
	micro.RegisterTTL(time.Second*30),
	micro.RegisterInterval(time.Second*15),
)
```
