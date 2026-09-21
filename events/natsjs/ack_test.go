package natsjs_test

import (
	"testing"
	"time"

	nserver "github.com/nats-io/nats-server/v2/server"
	nats "github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
	"go-micro.dev/v6/events"
	"go-micro.dev/v6/events/natsjs"
)

func TestManualAcknowledgements(t *testing.T) {
	srv, err := nserver.NewServer(&nserver.Options{Host: "127.0.0.1", Port: -1, JetStream: true, StoreDir: t.TempDir()})
	require.NoError(t, err)
	go srv.Start()
	require.True(t, srv.ReadyForConnections(5*time.Second))
	t.Cleanup(func() { srv.Shutdown(); srv.WaitForShutdown() })
	conn, err := nats.Connect(srv.ClientURL())
	require.NoError(t, err)
	t.Cleanup(conn.Close)
	js, err := conn.JetStream()
	require.NoError(t, err)
	client, err := natsjs.NewStream(natsjs.Address(srv.ClientURL()), natsjs.SynchronousPublish(true))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.(interface{ Close() error }).Close()) })
	ch, err := client.Consume("manual", events.WithGroup("worker"), events.WithAutoAck(false, 200*time.Millisecond))
	require.NoError(t, err)
	require.NoError(t, client.Publish("manual", []byte("payload")))
	receive := func() events.Event {
		t.Helper()
		select {
		case event := <-ch:
			return event
		case <-time.After(5 * time.Second):
			t.Fatal("expected event delivery")
			return events.Event{}
		}
	}
	first := receive()
	// Returning from the delivery callback must not implicitly acknowledge.
	second := receive()
	require.Equal(t, first.ID, second.ID)
	require.NoError(t, second.Nack())
	third := receive()
	require.Equal(t, first.ID, third.ID)
	require.NoError(t, third.Ack())
	require.Eventually(t, func() bool {
		info, err := js.ConsumerInfo("manual", "worker")
		return err == nil && info.NumAckPending == 0 && info.AckFloor.Stream == 1
	}, 5*time.Second, 10*time.Millisecond)
	select {
	case <-ch:
		t.Fatal("acknowledged event was redelivered")
	case <-time.After(400 * time.Millisecond):
	}
}
