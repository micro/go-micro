package cmd

import (
	grpcCli "github.com/asim/go-micro/plugins/client/grpc/v4"
	grpcSvr "github.com/asim/go-micro/plugins/server/grpc/v4"
	"go-micro.dev/v4/client"
	"go-micro.dev/v4/server"
)

// setupDefaults sets the default auth, broker etc implementations incase they arent configured by
// a profile. The default implementations are always the RPC implementations.
func setupDefaults() {
	client.DefaultClient = grpcCli.NewClient()
	server.DefaultServer = grpcSvr.NewServer()
}
