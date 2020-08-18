Prometheus
==========

A Prometheus "pull" based implementation of the metrics Reporter interface.


Capabilities
------------

* Go runtime metrics are handled natively by the Prometheus client library (CPU / MEM / GC / GoRoutines etc).
* User-defined metrics are registered in the Prometheus client dynamically (they must be pre-registered, hence all of the faffing around in metric_family.go).
* The metrics are made available on a Prometheus-compatible HTTP endpoint, which can be scraped at any time. This means that the user can very easily access stats even running locally as a standalone binary.
* Requires a micro.Server parameter (from which it gathers the service name and version). These are included as tags with every metric.


Usage
-----

```golang
    prometheusReporter := metrics.New(server)
    tags := metrics.Tags{"greeter": "Janos"}
    err := prometheusReporter.Count("hellos", 1, tags)
    if err != nil {
        fmt.Printf("Error setting a Count metric: %v", err)
    }
```
