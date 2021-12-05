package profile

import (
	"fmt"

	grpcCli "github.com/asim/go-micro/plugins/client/grpc/v4"
	grpcSvr "github.com/asim/go-micro/plugins/server/grpc/v4"
	"github.com/urfave/cli/v2"
	"go-micro.dev/v4/client"
	"go-micro.dev/v4/server"
)

// profiles which when called will configure micro to run in that environment
var profiles = map[string]*Profile{
	// built in profiles
	"client":     Client,
	"service":    Service,
	"test":       Test,
	"local":      Local,
	"kubernetes": Kubernetes,
}

// Profile configures an environment
type Profile struct {
	// name of the profile
	Name string
	// function used for setup
	Setup func(*cli.Context) error
	// TODO: presetup dependencies
	// e.g start resources
}

// Register a profile
func Register(name string, p *Profile) error {
	if _, ok := profiles[name]; ok {
		return fmt.Errorf("profile %s already exists", name)
	}
	profiles[name] = p
	return nil
}

// Load a profile
func Load(name string) (*Profile, error) {
	v, ok := profiles[name]
	if !ok {
		return nil, fmt.Errorf("profile %s does not exist", name)
	}
	return v, nil
}

// Client profile is for any entrypoint that behaves as a client
var Client = &Profile{
	Name: "client",
	Setup: func(ctx *cli.Context) error {
		SetupClient(grpcCli.NewClient())
		SetupServer(grpcSvr.NewServer())
		return nil
	},
}

// Service is the default for any services run
var Service = &Profile{
	Name:  "service",
	Setup: func(ctx *cli.Context) error { return nil },
}

// Local profile to run locally
var Local = &Profile{
	Name: "local",
	Setup: func(ctx *cli.Context) error {
		SetupClient(grpcCli.NewClient())
		SetupServer(grpcSvr.NewServer())
		return nil
	},
}

// Kubernetes profile to run on kubernetes with zero deps. Designed for use with the micro helm chart
var Kubernetes = &Profile{
	Name: "kubernetes",
	Setup: func(ctx *cli.Context) (err error) {

		return nil
	},
}

// Test profile is used for the go test suite
var Test = &Profile{
	Name: "test",
	Setup: func(ctx *cli.Context) error {

		return nil
	},
}

// SetupClient configures the default client
func SetupClient(c client.Client) {
	client.DefaultClient = c
}

// SetupServer configures the default server
func SetupServer(s server.Server) {
	server.DefaultServer = s
}
