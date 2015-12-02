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

// publishes a message
func pub(i int) {
	msg := client.NewPublication("topic.go.micro.srv.example", &example.Message{
		Say: fmt.Sprintf("This is a publication %d", i),
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

	fmt.Printf("Published %d: %v\n", i, msg)
}

func main() {
	cmd.Init()
	fmt.Println("\n--- Publisher example ---\n")
	//	for i := 0; i < 10; i++ {
	i := 0
	for {
		pub(i)
		i++
		time.Sleep(time.Millisecond * 5)
	}
}
