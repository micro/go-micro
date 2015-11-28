package main

import (
	"fmt"
	"time"

	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/cmd"
	c "github.com/micro/go-micro/context"
	example "github.com/micro/go-micro/examples/server/proto/example"
	"golang.org/x/net/context"
)

// wrapper example code

// log wrapper logs every time a request is made
type logWrapper struct {
	client.Client
}

func (l *logWrapper) Call(ctx context.Context, req client.Request, rsp interface{}) error {
	md, _ := c.GetMetadata(ctx)
	fmt.Printf("[Log Wrapper] ctx: %v service: %s method: %s\n", md, req.Service(), req.Method())
	return l.Client.Call(ctx, req, rsp)
}

// trace wrapper attaches a unique trace ID - timestamp
type traceWrapper struct {
	client.Client
}

func (t *traceWrapper) Call(ctx context.Context, req client.Request, rsp interface{}) error {
	ctx = c.WithMetadata(ctx, map[string]string{
		"X-Trace-Id": fmt.Sprintf("%d", time.Now().Unix()),
	})
	return t.Client.Call(ctx, req, rsp)
}

// Implements client.Wrapper as logWrapper
func logWrap(c client.Client) client.Client {
	return &logWrapper{c}
}

// Implements client.Wrapper as traceWrapper
func traceWrap(c client.Client) client.Client {
	return &traceWrapper{c}
}

// publishes a message
func pub() {
	msg := client.NewPublication("topic.go.micro.srv.example", &example.Message{
		Say: "This is a publication",
	})

	// create context with metadata
	ctx := c.WithMetadata(context.Background(), map[string]string{
		"X-User-Id": "john",
		"X-From-Id": "script",
	})

	// publish message
	if err := client.Publish(ctx, msg); err != nil {
		fmt.Println("pub err: ", err)
		return
	}

	fmt.Printf("Published: %v\n", msg)
}

func call(i int) {
	// Create new request to service go.micro.srv.example, method Example.Call
	req := client.NewRequest("go.micro.srv.example", "Example.Call", &example.Request{
		Name: "John",
	})

	// create context with metadata
	ctx := c.WithMetadata(context.Background(), map[string]string{
		"X-User-Id": "john",
		"X-From-Id": "script",
	})

	rsp := &example.Response{}

	// Call service
	if err := client.Call(ctx, req, rsp); err != nil {
		fmt.Println("call err: ", err, rsp)
		return
	}

	fmt.Println("Call:", i, "rsp:", rsp.Msg)
}

func stream() {
	// Create new request to service go.micro.srv.example, method Example.Call
	req := client.NewRequest("go.micro.srv.example", "Example.Stream", &example.StreamingRequest{
		Count: int64(10),
	})

	rspChan := make(chan *example.StreamingResponse, 10)

	stream, err := client.Stream(context.Background(), req, rspChan)
	if err != nil {
		fmt.Println("err:", err)
		return
	}

	for rsp := range rspChan {
		fmt.Println("Stream: rsp:", rsp.Count)
	}

	if stream.Error() != nil {
		fmt.Println("stream err:", err)
		return
	}

	if err := stream.Close(); err != nil {
		fmt.Println("stream close err:", err)
	}
}

func main() {
	cmd.Init()

	//	client.DefaultClient = client.NewClient(
	//		client.Codec("application/pb", pb.Codec),
	//		client.ContentType("application/pb"),
	//	)
	for {
		fmt.Println("\n--- Call example ---\n")
		for i := 0; i < 10; i++ {
			call(i)
		}

		fmt.Println("\n--- Streamer example ---\n")
		stream()

		fmt.Println("\n--- Publisher example ---\n")
		pub()

		fmt.Println("\n--- Wrapper example ---\n")

		// Wrap the default client
		client.DefaultClient = logWrap(client.DefaultClient)

		call(0)

		// Wrap using client.Wrap option
		client.DefaultClient = client.NewClient(
			client.Wrap(traceWrap),
			client.Wrap(logWrap),
		)

		call(1)
		time.Sleep(time.Millisecond * 100)
	}
}
