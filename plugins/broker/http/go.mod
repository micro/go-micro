module github.com/asim/go-micro/plugins/broker/http/v3

go 1.16

require (
	github.com/asim/go-micro/plugins/registry/memory/v3 v3.0.0-20210629124054-4929a7c16ecc
	github.com/asim/go-micro/v3 v3.5.2-0.20210629124054-4929a7c16ecc
	github.com/google/uuid v1.2.0
	golang.org/x/net v0.0.0-20210510120150-4163338589ed
)

replace (
	github.com/asim/go-micro/plugins/registry/memory/v3 => ../../../plugins/registry/memory
	github.com/asim/go-micro/v3 => ../../../../go-micro
)
