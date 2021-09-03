package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/asim/go-micro/v3"
	"github.com/asim/go-micro/v3/client"
	"github.com/urfave/cli/v2"
)

// Server sends a single client request and prints the server stream responses
// it receives. Exits on error.
func Server(ctx *cli.Context) error {
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

	var r interface{}
	request := c.NewRequest(service, endpoint, r, client.WithContentType("application/json"))

	stream, err := c.Stream(context.Background(), request)
	if err != nil {
		return err
	}
	if err := stream.Send(creq); err != nil {
		return err
	}

	for stream.Error() == nil {
		rsp := &map[string]interface{}{}
		err := stream.Recv(rsp)
		if err == io.EOF {
			return nil
		}
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
