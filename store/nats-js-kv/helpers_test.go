package natsjskv

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	nserver "github.com/nats-io/nats-server/v2/server"
	"github.com/pkg/errors"
	"github.com/test-go/testify/require"
	"go-micro.dev/v5/store"
)

func testSetup(ctx context.Context, t *testing.T, opts ...store.Option) store.Store {
	t.Helper()

	var err error
	var s store.Store
	for i := 0; i < 5; i++ {
		nCtx, cancel := context.WithCancel(ctx)
		addr := startNatsServer(nCtx, t)

		opts = append(opts, store.Nodes(addr), EncodeKeys())
		s = NewStore(opts...)

		err = s.Init()
		if err != nil {
			t.Log(errors.Wrap(err, "Error: Server initialization failed, restarting server"))
			cancel()
			if err = s.Close(); err != nil {
				t.Logf("Failed to close store: %v", err)
			}
			time.Sleep(time.Second)
			continue
		}

		go func() {
			<-ctx.Done()
			cancel()
			if err = s.Close(); err != nil {
				t.Logf("Failed to close store: %v", err)
			}
		}()

		return s
	}
	t.Error(errors.Wrap(err, "Store initialization failed"))
	return s
}

func startNatsServer(ctx context.Context, t *testing.T) string {
	t.Helper()
	natsAddr := getFreeLocalhostAddress()
	natsPort, err := strconv.Atoi(strings.Split(natsAddr, ":")[1])
	if err != nil {
		t.Logf("Failed to parse port from address: %v", err)
	}

	clusterName := "gomicro-store-test-cluster"

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

	time.Sleep(2 * time.Second)

	return natsAddr
}

func getFreeLocalhostAddress() string {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return ""
	}

	addr := l.Addr().String()
	if err := l.Close(); err != nil {
		return addr
	}
	return addr
}

func natsServer(ctx context.Context, t *testing.T, opts *nserver.Options) {
	t.Helper()

	opts.TLSTimeout = 180
	server, err := nserver.NewServer(
		opts,
	)
	require.NoError(t, err)
	if err != nil {
		return
	}
	defer server.Shutdown()

	server.SetLoggerV2(
		NewLogWrapper(),
		false, false, false,
	)

	tmpdir := t.TempDir()
	natsdir := filepath.Join(tmpdir, "nats-js")
	jsConf := &nserver.JetStreamConfig{
		StoreDir: natsdir,
	}

	// first start NATS
	go server.Start()
	time.Sleep(time.Second)

	// second start JetStream
	err = server.EnableJetStream(jsConf)
	require.NoError(t, err)
	if err != nil {
		return
	}

	// This fixes some issues where tests fail because directory cleanup fails
	t.Cleanup(func() {
		contents, err := filepath.Glob(natsdir + "/*")
		if err != nil {
			t.Logf("Failed to glob directory: %v", err)
		}
		for _, item := range contents {
			if err := os.RemoveAll(item); err != nil {
				t.Logf("Failed to remove file: %v", err)
			}
		}
		if err := os.RemoveAll(natsdir); err != nil {
			t.Logf("Failed to remove directory: %v", err)
		}
	})

	<-ctx.Done()
}

func NewLogWrapper() *LogWrapper {
	return &LogWrapper{}
}

type LogWrapper struct {
}

// Noticef logs a notice statement.
func (l *LogWrapper) Noticef(_ string, _ ...interface{}) {
}

// Warnf logs a warning statement.
func (l *LogWrapper) Warnf(format string, v ...interface{}) {
	fmt.Printf(format+"\n", v...)
}

// Fatalf logs a fatal statement.
func (l *LogWrapper) Fatalf(format string, v ...interface{}) {
	fmt.Printf(format+"\n", v...)
}

// Errorf logs an error statement.
func (l *LogWrapper) Errorf(format string, v ...interface{}) {
	fmt.Printf(format+"\n", v...)
}

// Debugf logs a debug statement.
func (l *LogWrapper) Debugf(_ string, _ ...interface{}) {
}

// Tracef logs a trace statement.
func (l *LogWrapper) Tracef(format string, v ...interface{}) {
	fmt.Printf(format+"\n", v...)
}
