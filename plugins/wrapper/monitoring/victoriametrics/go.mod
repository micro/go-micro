module github.com/asim/go-micro/plugins/wrapper/monitoring/victoriametrics/v3

go 1.16

require (
	github.com/VictoriaMetrics/metrics v1.17.2
	github.com/asim/go-micro/plugins/registry/memory/v3 v3.0.0-20210629124054-4929a7c16ecc
	github.com/asim/go-micro/v3 v3.5.2-0.20210629124054-4929a7c16ecc
	github.com/stretchr/testify v1.7.0
)

replace (
	github.com/asim/go-micro/plugins/registry/memory/v3 => ../../../../plugins/registry/memory
	github.com/asim/go-micro/v3 => ../../../../../go-micro
)
