module github.com/asim/go-micro/plugins/codec/segmentio/v4

go 1.17

require (
	github.com/golang/protobuf v1.5.2
	github.com/oxtoacart/bpool v0.0.0-20190530202638-03653db5a59c
	github.com/segmentio/encoding v0.1.12
	go-micro.dev/v4 v4.2.1
)

require (
	github.com/google/go-cmp v0.5.6 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/protobuf v1.26.0 // indirect
)

replace go-micro.dev/v4 => ../../../../go-micro
