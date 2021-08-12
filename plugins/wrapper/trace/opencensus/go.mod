module github.com/asim/go-micro/plugins/wrapper/trace/opencensus/v3

go 1.16

require (
	github.com/asim/go-micro/v3 v3.5.2-0.20210630062103-c13bb07171bc
	go.opencensus.io v0.23.0
	google.golang.org/genproto v0.0.0-20210624195500-8bfb893ecb84
)

replace github.com/asim/go-micro/v3 => ../../../../../go-micro
