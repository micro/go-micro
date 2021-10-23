module github.com/asim/go-micro/plugins/wrapper/monitoring/victoriametrics/v4

go 1.16

require (
	github.com/VictoriaMetrics/metrics v1.17.2
	github.com/stretchr/testify v1.7.0
	go-micro.dev/v4 v4.2.1
)

replace go-micro.dev/v4 => ../../../../../go-micro
