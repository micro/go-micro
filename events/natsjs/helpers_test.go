package natsjs_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	nserver "github.com/nats-io/nats-server/v2/server"
)

func getFreeLocalhostAddress() string {
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	defer l.Close()
	return l.Addr().String()
}

func natsServer(ctx context.Context, t *testing.T, opts *nserver.Options) {
	t.Helper()

	// Report errors with Errorf (not Fatalf/require), which are safe to
	// call from this non-test goroutine; Fatalf/FailNow are not.
	server, err := nserver.NewServer(opts)
	if err != nil {
		t.Errorf("nats: new server: %v", err)
		return
	}

	server.SetLoggerV2(
		NewLogWrapper(),
		true, true, false,
	)

	// first start NATS
	go server.Start()
	if !server.ReadyForConnections(time.Second * 10) {
		t.Errorf("NATS server not ready")
		return
	}

	// Manage the JetStream store dir ourselves rather than via t.TempDir.
	// t.TempDir registers a RemoveAll that runs when the test ends, which
	// races this goroutine's shutdown — the server can still be releasing
	// JetStream files, leaving the dir non-empty ("directory not empty").
	// Remove it here instead, only after the server has fully stopped.
	storeDir, err := os.MkdirTemp("", "nats-js")
	if err != nil {
		t.Errorf("nats: temp dir: %v", err)
		return
	}
	defer os.RemoveAll(storeDir)

	// second start JetStream
	if err := server.EnableJetStream(&nserver.JetStreamConfig{StoreDir: filepath.Join(storeDir, "nats-js")}); err != nil {
		t.Errorf("nats: enable jetstream: %v", err)
		return
	}

	<-ctx.Done()

	server.Shutdown()
	server.WaitForShutdown()
}

func NewLogWrapper() *LogWrapper {
	return &LogWrapper{}
}

type LogWrapper struct {
}

// Noticef logs a notice statement.
func (l *LogWrapper) Noticef(format string, v ...interface{}) {
	fmt.Printf(format+"\n", v...)
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
func (l *LogWrapper) Debugf(format string, v ...interface{}) {
	fmt.Printf(format+"\n", v...)
}

// Tracef logs a trace statement.
func (l *LogWrapper) Tracef(format string, v ...interface{}) {
	fmt.Printf(format+"\n", v...)
}
