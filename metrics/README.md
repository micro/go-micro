metrics
=======

The metrics package provides a simple metrics "Reporter" interface which allows the user to submit counters, gauges and timings (along with key/value tags).

Implementations
---------------

* Prometheus (pull): will be first
* Prometheus (push): certainly achievable
* InfluxDB: could quite easily be done
* Telegraf: almost identical to the InfluxDB implementation
* Micro: Could we provide metrics over Micro's server interface?


Todo
----

* Include a handler middleware which uses the Reporter interface to generate per-request level metrics
    - Throughput
    - Errors
    - Duration
