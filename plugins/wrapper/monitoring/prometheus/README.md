# Prometheus 

Wrappers are a form of middleware that can be used with go-micro services. They can wrap both the Client and Server handlers. 
This plugin implements the HandlerWrapper interface to provide automatic prometheus metric handling
for each microservice method execution time and operation count for success and failed cases.  

This handler will export two metrics to prometheus:
* **micro_request_total**. How many go-micro requests processed, partitioned by method and status.
* **micro_request_duration_microseconds**. Service method request latencies in microseconds, partitioned by method.

# Usage

When creating your service, add the wrapper like so.

```go
    service := micro.NewService(
        micro.Name("service name"),
    	micro.Version("latest"),
    	micro.WrapHandler(prometheus.NewHandlerWrapper()),
    )
    
    service.Init()
```

