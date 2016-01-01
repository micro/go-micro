package main

import (
	"fmt"
	"os"

	"github.com/micro/cli"
	micro "github.com/micro/go-micro"
	proto "github.com/micro/go-micro/examples/service/proto"
	"golang.org/x/net/context"
)

/*

Example usage of top level service initialisation

*/

type Greeter struct{}

func (g *Greeter) Hello(ctx context.Context, req *proto.HelloRequest, rsp *proto.HelloResponse) error {
	rsp.Greeting = "Hello " + req.Name
	return nil
}

// Setup and the client
func client(service micro.Service) {
	// Create new greeter client
	greeter := proto.NewGreeterClient("greeter", service.Client())

	// Call the greeter
	rsp, err := greeter.Hello(context.TODO(), &proto.HelloRequest{Name: "John"})
	if err != nil {
		fmt.Println(err)
	}

	// Print response
	fmt.Println(rsp.Greeting)
}

// Setup some command line flags
func flags(service micro.Service) {
	app := service.Cmd().App()
	app.Flags = append(app.Flags,
		&cli.BoolFlag{
			Name:  "server",
			Usage: "Launch the server",
		},
		&cli.BoolFlag{
			Name:  "client",
			Usage: "Launch the client",
		},
	)

	// Let's launch the server or the client
	app.Action = func(c *cli.Context) {
		if c.Bool("client") {
			client(service)
			os.Exit(0)
		}
	}
}

func main() {
	// Create a new service. Optionally include some options here.
	service := micro.NewService(
		micro.Name("greeter"),
		micro.Version("latest"),
		micro.Metadata(map[string]string{
			"type": "helloworld",
		}),
	)

	// Setup some flags. Specify --client to run the client
	flags(service)

	// Init will parse the command line flags. Any flags set will
	// override the above settings. Options defined here will
	// override anything set on the command line.
	service.Init()

	// By default we'll run the server unless the flags catch us

	// Setup the server

	// Register handler
	proto.RegisterGreeterHandler(service.Server(), new(Greeter))

	// Run the server
	if err := service.Run(); err != nil {
		fmt.Println(err)
	}
}
