module github.com/asim/go-micro/plugins/store/mysql/v4

go 1.16

require (
	github.com/go-sql-driver/mysql v1.6.0
	github.com/pkg/errors v0.9.1
	go-micro.dev/v4 v4.1.0
)

replace go-micro.dev/v4 => ../../../../go-micro
