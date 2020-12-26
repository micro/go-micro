# Static selector

The Static selector returns the named service as a node for every request. This is useful where you want to 
offload discovery and balancing to DNS (particularly useful with KubeDNS).

This DOES however require a static port assignment (because we no longer have the ability to look up metadata). This defaults to port 8080, but can be overriddden at runtime using env-vars.

An optional domain-name can be appended too.


## Environment variables

* "STATIC_SELECTOR_DOMAIN_NAME": An optional domain-name to append to the speicified service name.
* "STATIC_SELECTOR_PORT_NUMBER": Override the default port (8080) for "discovered" services.


## Usage

```go
selector := static.NewSelector()

service := micro.NewService(
	client.NewClient(client.Selector(selector))
)
```
