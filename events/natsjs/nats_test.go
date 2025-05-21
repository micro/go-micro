package natsjs_test

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"go-micro.dev/v5/events/natsjs"
	nserver "github.com/nats-io/nats-server/v2/server"
	"github.com/stretchr/testify/assert"
	"github.com/test-go/testify/require"
	"go-micro.dev/v5/events"
)

type Payload struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func TestSingleEvent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	// variables
	demoPayload := Payload{
		ID:   "123",
		Name: "Hello World",
	}
	topic := "foobar"

	clusterName := "test-cluster"

	natsAddr := getFreeLocalhostAddress()
	natsPort, _ := strconv.Atoi(strings.Split(natsAddr, ":")[1])

	// start the NATS with JetStream server
	go natsServer(ctx,
		t,
		&nserver.Options{
			Host: strings.Split(natsAddr, ":")[0],
			Port: natsPort,
			Cluster: nserver.ClusterOpts{
				Name: clusterName,
			},
		},
	)

	time.Sleep(1 * time.Second)

	// consumer
	consumerClient, err := natsjs.NewStream(
		natsjs.Address(natsAddr),
		natsjs.ClusterID(clusterName),
	)
	require.NoError(t, err)
	if err != nil {
		return
	}

	consumer := func(_ context.Context, t *testing.T, client events.Stream, cancel context.CancelFunc) {
		t.Helper()
		defer cancel()

		foobarEvents, err := client.Consume(topic)
		require.Nil(t, err)
		if err != nil {
			return
		}

		// wait for the event
		event := <-foobarEvents

		p := Payload{}
		err = json.Unmarshal(event.Payload, &p)

		require.NoError(t, err)
		if err != nil {
			return
		}

		assert.Equal(t, demoPayload.ID, p.ID)
		assert.Equal(t, demoPayload.Name, p.Name)
	}

	go consumer(ctx, t, consumerClient, cancel)

	// publisher
	time.Sleep(1 * time.Second)

	publisherClient, err := natsjs.NewStream(
		natsjs.Address(natsAddr),
		natsjs.ClusterID(clusterName),
	)
	require.NoError(t, err)
	if err != nil {
		return
	}

	publisher := func(_ context.Context, t *testing.T, client events.Stream) {
		t.Helper()
		err := client.Publish(topic, demoPayload)
		require.NoError(t, err)
	}

	go publisher(ctx, t, publisherClient)

	// wait until consumer received the event
	<-ctx.Done()
}
