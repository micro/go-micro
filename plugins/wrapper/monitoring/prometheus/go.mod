module github.com/asim/go-micro/plugins/wrapper/monitoring/prometheus/v4

go 1.16

require (
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/client_model v0.2.0
	github.com/stretchr/testify v1.7.0
	go-micro.dev/v4 v4.2.1
)

replace go-micro.dev/v4 => ../../../../../go-micro
