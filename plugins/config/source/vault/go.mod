module github.com/asim/go-micro/plugins/config/source/vault/v4

go 1.16

require (
	github.com/hashicorp/vault/api v1.0.4
	go-micro.dev/v4 v4.1.0
)

replace go-micro.dev/v4 => ../../../../../go-micro
