module github.com/asim/go-micro/plugins/wrapper/monitoring/victoriametrics/v4

go 1.16

require (
	github.com/VictoriaMetrics/metrics v1.17.2
	github.com/asim/go-micro/plugins/registry/memory/v4 master
	go-micro.dev/v4 v4.1.0
	github.com/stretchr/testify v1.7.0
)

replace (
	github.com/asim/go-micro/plugins/registry/memory/v4 => ../../../../plugins/registry/memory
	go-micro.dev/v4 => ../../../../../go-micro
)
