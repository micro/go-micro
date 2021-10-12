module github.com/asim/go-micro/plugins/wrapper/trace/opencensus/v3

go 1.16

require (
	go-micro.dev/v4 v4.1.0
	go.opencensus.io v0.23.0
	google.golang.org/genproto v0.0.0-20210624195500-8bfb893ecb84
)

replace github.com/asim/go-micro/v3 => ../../../../../go-micro
