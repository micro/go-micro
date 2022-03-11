package natsjs_test

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/asim/go-micro/plugins/events/natsjs/v4"
	nserver "github.com/nats-io/nats-server/v2/server"
	"github.com/stretchr/testify/assert"
	"go-micro.dev/v4/events"
)

type Payload struct {
	ID   string
	Name string
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
	assert.NoError(t, err)
	if err != nil {
		return
	}

	consumer := func(ctx context.Context, t *testing.T, client events.Stream, cancel context.CancelFunc) {
		defer cancel()

		foobarEvents, err := client.Consume(topic)
		assert.Nil(t, err)
		if err != nil {
			return
		}

		// wait for the event
		event := <-foobarEvents

		p := Payload{}
		err = json.Unmarshal(event.Payload, &p)

		assert.NoError(t, err)
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
	assert.NoError(t, err)
	if err != nil {
		return
	}

	publisher := func(ctx context.Context, t *testing.T, client events.Stream) {
		err := client.Publish(topic, demoPayload)
		assert.NoError(t, err)
	}

	go publisher(ctx, t, publisherClient)

	// wait until consumer received the event
	<-ctx.Done()
}
