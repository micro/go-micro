package call

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/asim/go-micro/cmd/gomu/cmd"
	"github.com/asim/go-micro/v3"
	"github.com/asim/go-micro/v3/client"
	"github.com/urfave/cli/v2"
)

func init() {
	cmd.Register(&cli.Command{
		Name:   "call",
		Usage:  "Call a service, e.g. " + cmd.App().Name + " call helloworld Helloworld.Call '{\"name\": \"John\"}'",
		Action: RunCall,
	})
}

// RunCall calls a service endpoint and prints its response. Exits on error.
func RunCall(ctx *cli.Context) error {
	args := ctx.Args().Slice()
	if len(args) < 2 {
		return cli.ShowSubcommandHelp(ctx)
	}

	service := args[0]
	endpoint := args[1]
	req := strings.Join(args[2:], " ")
	if len(req) == 0 {
		req = `{}`
	}

	d := json.NewDecoder(strings.NewReader(req))
	d.UseNumber()

	var creq map[string]interface{}
	if err := d.Decode(&creq); err != nil {
		return err
	}

	srv := micro.NewService()
	srv.Init()
	c := srv.Client()

	request := c.NewRequest(service, endpoint, creq, client.WithContentType("application/json"))
	var response map[string]interface{}

	if err := c.Call(context.Background(), request, &response); err != nil {
		return err
	}

	b, err := json.Marshal(response)
	if err != nil {
		return err
	}

	fmt.Println(string(b))
	return nil
}
