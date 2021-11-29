module github.com/asim/go-micro/plugins/logger/zap/v4

go 1.17

require (
	go-micro.dev/v4 v4.2.1
	go.uber.org/zap v1.17.0
)

require (
	github.com/google/uuid v1.2.0 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
)

replace go-micro.dev/v4 => ../../../../go-micro
