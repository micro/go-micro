package microcli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v5/client"
	"go-micro.dev/v5/cmd"
	"go-micro.dev/v5/codec/bytes"
	"go-micro.dev/v5/registry"

	"go-micro.dev/v5/cmd/micro/cli/new"
	"go-micro.dev/v5/cmd/micro/cli/util"

	// Import packages that register commands via init()
	_ "go-micro.dev/v5/cmd/micro/cli/build"
	_ "go-micro.dev/v5/cmd/micro/cli/deploy"
	_ "go-micro.dev/v5/cmd/micro/cli/init"
	_ "go-micro.dev/v5/cmd/micro/cli/remote"
)

var (
	// version is set by the release action
	// this is the default for local builds
	version = "5.0.0-dev"
)

func genProtoHandler(c *cli.Context) error {
	cmd := exec.Command("find", ".", "-name", "*.proto", "-exec", "protoc", "--proto_path=.", "--micro_out=.", "--go_out=.", `{}`, `;`)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func init() {
	cmd.Register([]*cli.Command{
		{
			Name:   "new",
			Usage:  "Create a new service",
			Action: new.Run,
		},
		{
			Name:  "gen",
			Usage: "Generate various things",
			Subcommands: []*cli.Command{
				{
					Name:   "proto",
					Usage:  "Generate proto requires protoc and protoc-gen-micro",
					Action: genProtoHandler,
				},
			},
		},
		{
			Name:  "services",
			Usage: "List available services",
			Action: func(ctx *cli.Context) error {
				services, err := registry.ListServices()
				if err != nil {
					return err
				}
				for _, service := range services {
					fmt.Println(service.Name)
				}
				return nil
			},
		},
		{
			Name:  "call",
			Usage: "Call a service",
			Flags: []cli.Flag{
				&cli.StringSliceFlag{
					Name:    "header",
					Aliases: []string{"H"},
					Usage:   "Set request headers (can be used multiple times): --header 'Key:Value'",
				},
				&cli.StringSliceFlag{
					Name:    "metadata",
					Aliases: []string{"m"},
					Usage:   "Set request metadata (can be used multiple times): --metadata 'Key:Value'",
				},
			},
			Action: func(ctx *cli.Context) error {
				args := ctx.Args()

				if args.Len() < 2 {
					return fmt.Errorf("Usage: [service] [endpoint] [request]")
				}

				service := args.Get(0)
				endpoint := args.Get(1)
				request := `{}`

				if args.Len() == 3 {
					request = args.Get(2)
				}

				// Create context with metadata if provided
				// Note: This is for the direct 'micro call' command.
				// Dynamic service calls (e.g., 'micro helloworld call') are handled in CallService.
				callCtx := context.TODO()
				callCtx = util.AddMetadataToContext(callCtx, ctx.StringSlice("metadata"))
				callCtx = util.AddMetadataToContext(callCtx, ctx.StringSlice("header"))

				req := client.NewRequest(service, endpoint, &bytes.Frame{Data: []byte(request)})
				var rsp bytes.Frame
				err := client.Call(callCtx, req, &rsp)
				if err != nil {
					return err
				}

				fmt.Print(string(rsp.Data))
				return nil
			},
		},
		{
			Name:  "describe",
			Usage: "Describe a service",
			Action: func(ctx *cli.Context) error {
				args := ctx.Args()

				if args.Len() != 1 {
					return fmt.Errorf("Usage: [service]")
				}

				service := args.Get(0)
				services, err := registry.GetService(service)
				if err != nil {
					return err
				}
				if len(services) == 0 {
					return nil
				}
				b, _ := json.MarshalIndent(services[0], "", "    ")
				fmt.Println(string(b))
				return nil
			},
		},
		// Note: The following commands are registered in their respective packages:
		// - status, logs, stop: remote/remote.go
		// - build: build/build.go
		// - deploy: deploy/deploy.go
		// - init: init/init.go
	}...)

	cmd.App().Action = func(c *cli.Context) error {
		if c.Args().Len() == 0 {
			return nil
		}

		v, err := exec.LookPath("micro-" + c.Args().First())
		if err == nil {
			ce := exec.Command(v, c.Args().Slice()[1:]...)
			ce.Stdout = os.Stdout
			ce.Stderr = os.Stderr
			return ce.Run()
		}

		command := c.Args().Get(0)
		args := c.Args().Slice()

		if srv, err := util.LookupService(command); err != nil {
			return util.CliError(err)
		} else if srv != nil && util.ShouldRenderHelp(args) {
			return cli.Exit(util.FormatServiceUsage(srv, c), 0)
		} else if srv != nil {
			err := util.CallService(srv, args)
			return util.CliError(err)
		}

		return nil
	}
}
