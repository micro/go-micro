module github.com/asim/go-micro/plugins/logger/logrus/v4

go 1.17

require (
	github.com/sirupsen/logrus v1.8.1
	go-micro.dev/v4 v4.2.1
)

require (
	github.com/google/uuid v1.2.0 // indirect
	golang.org/x/sys v0.0.0-20210502180810-71e4cd670f79 // indirect
)

replace go-micro.dev/v4 => ../../../../go-micro
