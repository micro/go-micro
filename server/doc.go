/*
Package server is an interface for a micro server.

It represents a server instance in go-micro which handles synchronous
requests via handlers and asynchronous requests via subscribers that
register with a broker.

The server combines the all the packages in go-micro to create a whole unit
used for building applications including discovery, client/server communication
and pub/sub.

        import "github.com/micro/go-micro/server"

        type Greeter struct {}

        func (g *Greeter) Hello(ctx context.Context, req *greeter.Request, rsp *greeter.Response) error {
                rsp.Msg = "Hello " + req.Name
                return nil
        }

        s := server.NewServer()


        s.Handle(
                s.NewHandler(&Greeter{}),
        )

        s.Start()

*/
package server
