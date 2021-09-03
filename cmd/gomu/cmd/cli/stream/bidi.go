package stream

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/asim/go-micro/v3"
	"github.com/asim/go-micro/v3/client"
	"github.com/urfave/cli/v2"
)

// Bidirectional streams client requests and prints the server stream responses
// it receives. Exits on error.
func Bidirectional(ctx *cli.Context) error {
	args := ctx.Args().Slice()
	if len(args) < 2 {
		return cli.ShowSubcommandHelp(ctx)
	}

	service := args[0]
	endpoint := args[1]
	requests := args[2:]

	srv := micro.NewService()
	srv.Init()
	c := srv.Client()

	var r interface{}
	request := c.NewRequest(service, endpoint, r, client.WithContentType("application/json"))
	var rsp map[string]interface{}
	stream, err := c.Stream(ctx.Context, request)
	if err != nil {
		return err
	}

	for _, req := range requests {
		d := json.NewDecoder(strings.NewReader(req))
		d.UseNumber()

		var creq map[string]interface{}
		if err := d.Decode(&creq); err != nil {
			return err
		}

		if err := stream.Send(creq); err != nil {
			return err
		}

		err := stream.Recv(&rsp)
		if err != nil {
			return err
		}

		b, err := json.Marshal(rsp)
		if err != nil {
			return err
		}
		fmt.Println(string(b))
	}
	if stream.Error() != nil {
		return stream.Error()
	}
	if err := stream.Close(); err != nil {
		return err
	}

	return nil
}
