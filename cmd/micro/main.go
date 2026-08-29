package main

import (
	"embed"

	"go-micro.dev/v6/cmd"
	builtin "go-micro.dev/v6/internal/otel"

	// Link the OTLP trace exporter so CLI-hosted services/agents/flows emit
	// spans when OTEL_EXPORTER_OTLP_ENDPOINT is set. Library users omit it.
	_ "go-micro.dev/v6/otel"

	// Link every plugin so CLI flag selection (--registry etcd, --broker nats,
	// --profile nats, ...) keeps working; library users omit this import.
	_ "go-micro.dev/v6/cmd/defaults"

	_ "go-micro.dev/v6/cmd/micro/a2a"
	_ "go-micro.dev/v6/cmd/micro/ai"
	_ "go-micro.dev/v6/cmd/micro/api"
	_ "go-micro.dev/v6/cmd/micro/chat"
	_ "go-micro.dev/v6/cmd/micro/cli"
	_ "go-micro.dev/v6/cmd/micro/cli/build"
	_ "go-micro.dev/v6/cmd/micro/cli/deploy"
	_ "go-micro.dev/v6/cmd/micro/flow"
	"go-micro.dev/v6/cmd/micro/gateway"
	_ "go-micro.dev/v6/cmd/micro/inspect"
	_ "go-micro.dev/v6/cmd/micro/loop"
	_ "go-micro.dev/v6/cmd/micro/mcp"
	_ "go-micro.dev/v6/cmd/micro/resource"
	_ "go-micro.dev/v6/cmd/micro/run"
)

//go:embed web/styles.css web/main.js web/templates/*
var webFS embed.FS

var version = "5.0.0-dev"

func init() {
	gateway.HTML = webFS
}

func main() {
	builtin.Init()
	_ = cmd.Init(
		cmd.Name("micro"),
		cmd.Version(version),
	)
}
