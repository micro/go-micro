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

func TestDottedSubjectsStableIDsAndSharedStream(t *testing.T) {
	for _, shared := range []bool{false, true} {
		name := "default"
		if shared {
			name = "shared"
		}
		t.Run(name, func(t *testing.T) {
			srv, err := nserver.NewServer(&nserver.Options{Host: "127.0.0.1", Port: -1, JetStream: true, StoreDir: t.TempDir()})
			require.NoError(t, err)
			go srv.Start()
			require.True(t, srv.ReadyForConnections(5*time.Second))
			defer func() { srv.Shutdown(); srv.WaitForShutdown() }()
			conn, err := nats.Connect(srv.ClientURL())
			require.NoError(t, err)
			defer conn.Close()
			js, err := conn.JetStream()
			require.NoError(t, err)
			opts := []natsjs.Option{natsjs.Address(srv.ClientURL()), natsjs.SynchronousPublish(true)}
			streamName := natsjs.StreamName("orders.created.v1")
			if shared {
				streamName = "orders"
				opts = append(opts, natsjs.WithStreamConfig(func(string) (nats.StreamConfig, error) {
					return nats.StreamConfig{Name: "orders", Subjects: []string{"orders.*.v1"}, Storage: nats.MemoryStorage, MaxMsgs: 100, Replicas: 1, Duplicates: time.Minute}, nil
				}))
			}
			client, err := natsjs.NewStream(opts...)
			require.NoError(t, err)
			defer client.(interface{ Close() error }).Close()
			// Publish before any consumer exists; retries preserve the ID and deduplicate.
			require.NoError(t, client.Publish("orders.created.v1", []byte("first"), events.WithID("outbox-1")))
			require.NoError(t, client.Publish("orders.created.v1", []byte("first"), events.WithID("outbox-1")))
			info, err := js.StreamInfo(streamName)
			require.NoError(t, err)
			require.EqualValues(t, 1, info.State.Msgs)
			if shared {
				require.Equal(t, nats.MemoryStorage, info.Config.Storage)
				require.EqualValues(t, 100, info.Config.MaxMsgs)
			}
			ch, err := client.Consume("orders.created.v1", events.WithGroup("created"))
			require.NoError(t, err)
			select {
			case event := <-ch:
				require.Equal(t, "outbox-1", event.ID)
				require.Equal(t, "orders.created.v1", event.Topic)
			case <-time.After(time.Second):
				t.Fatal("missing event")
			}
			if shared {
				require.NoError(t, client.Publish("orders.updated.v1", []byte("second"), events.WithID("outbox-2")))
				second, err := client.Consume("orders.updated.v1", events.WithGroup("updated"))
				require.NoError(t, err)
				select {
				case event := <-second:
					require.Equal(t, "outbox-2", event.ID)
				case <-time.After(time.Second):
					t.Fatal("missing second subject")
				}
			}
		})
	}
	require.Equal(t, "legacy", natsjs.StreamName("legacy"))
	require.NotEqual(t, natsjs.StreamName("orders.created.v1"), natsjs.StreamName("orders_created_v1"))
}
