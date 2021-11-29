module github.com/asim/go-micro/plugins/codec/msgpackrpc/v4

go 1.17

require (
	github.com/tinylib/msgp v1.1.6
	go-micro.dev/v4 v4.2.1
)

require github.com/philhofer/fwd v1.1.1 // indirect

replace go-micro.dev/v4 => ../../../../go-micro
