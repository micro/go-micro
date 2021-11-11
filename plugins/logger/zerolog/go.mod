module github.com/asim/go-micro/plugins/logger/zerolog/v4

go 1.17

require (
	github.com/rs/zerolog v1.23.0
	go-micro.dev/v4 v4.2.1
)

require (
	github.com/google/uuid v1.2.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
)

replace go-micro.dev/v4 => ../../../../go-micro
