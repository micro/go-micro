/*
Package client is an interface for making requests.

It provides a method to make synchronous, asynchronous and streaming requests to services.
By default json and protobuf codecs are supported.

        import "github.com/micro/go-micro/client"

        c := client.NewClient()

        req := c.NewRequest("go.micro.srv.greeter", "Greeter.Hello", &greeter.Request{
                Name: "John",
        })

        rsp := &greeter.Response{}

        if err := c.Call(context.Background(), req, rsp); err != nil {
                return err
        }

        fmt.Println(rsp.Msg)
*/
package client
