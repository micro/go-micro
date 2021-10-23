module github.com/asim/go-micro/plugins/wrapper/trace/opencensus/v4

go 1.16

require (
	go-micro.dev/v4 v4.2.1
	go.opencensus.io v0.23.0
	google.golang.org/genproto v0.0.0-20210624195500-8bfb893ecb84
)

replace go-micro.dev/v4 => ../../../../../go-micro
