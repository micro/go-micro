package natsjs_test

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"testing"

	nserver "github.com/nats-io/nats-server/v2/server"
	"github.com/stretchr/testify/assert"
)

func getFreeLocalhostAddress() string {
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	defer l.Close()
	return l.Addr().String()

}

func natsServer(ctx context.Context, t *testing.T, opts *nserver.Options) {
	server, err := nserver.NewServer(
		opts,
	)
	assert.NoError(t, err)
	if err != nil {
		return
	}

	server.SetLoggerV2(
		NewLogWrapper(),
		true, true, false,
	)

	// first start NATS
	go server.Start()

	jsConf := &nserver.JetStreamConfig{
		StoreDir: filepath.Join(t.TempDir(), "nats-js"),
	}

	// second start JetStream
	err = server.EnableJetStream(jsConf)
	assert.NoError(t, err)
	if err != nil {
		return
	}

	<-ctx.Done()

	server.Shutdown()
}

func NewLogWrapper() *LogWrapper {
	return &LogWrapper{}
}

type LogWrapper struct {
}

// Noticef logs a notice statement
func (l *LogWrapper) Noticef(format string, v ...interface{}) {
	fmt.Printf(format+"\n", v...)
}

// Warnf logs a warning statement
func (l *LogWrapper) Warnf(format string, v ...interface{}) {
	fmt.Printf(format+"\n", v...)
}

// Fatalf logs a fatal statement
func (l *LogWrapper) Fatalf(format string, v ...interface{}) {
	fmt.Printf(format+"\n", v...)
}

// Errorf logs an error statement
func (l *LogWrapper) Errorf(format string, v ...interface{}) {
	fmt.Printf(format+"\n", v...)
}

// Debugf logs a debug statement
func (l *LogWrapper) Debugf(format string, v ...interface{}) {
	fmt.Printf(format+"\n", v...)
}

// Tracef logs a trace statement
func (l *LogWrapper) Tracef(format string, v ...interface{}) {
	fmt.Printf(format+"\n", v...)
}
