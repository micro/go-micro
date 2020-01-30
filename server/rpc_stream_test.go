package server

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/micro/go-micro/v2/codec/json"
	protoCodec "github.com/micro/go-micro/v2/codec/proto"
)

// protoStruct implements proto.Message
type protoStruct struct {
	Payload string `protobuf:"bytes,1,opt,name=service,proto3" json:"service,omitempty"`
}

func (m *protoStruct) Reset()         { *m = protoStruct{} }
func (m *protoStruct) String() string { return proto.CompactTextString(m) }
func (*protoStruct) ProtoMessage()    {}

// safeBuffer throws away everything and wont Read data back
type safeBuffer struct {
	sync.RWMutex
	buf []byte
	off int
}

func (b *safeBuffer) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	// Cannot retain p, so we must copy it:
	p2 := make([]byte, len(p))
	copy(p2, p)
	b.Lock()
	b.buf = append(b.buf, p2...)
	b.Unlock()
	return len(p2), nil
}

func (b *safeBuffer) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	b.RLock()
	n = copy(p, b.buf[b.off:])
	b.RUnlock()
	if n == 0 {
		return 0, io.EOF
	}
	b.off += n
	return n, nil
}

func (b *safeBuffer) Close() error {
	return nil
}

func TestRPCStream_Sequence(t *testing.T) {
	buffer := new(bytes.Buffer)
	rwc := readWriteCloser{
		rbuf: buffer,
		wbuf: buffer,
	}
	codec := json.NewCodec(&rwc)
	streamServer := rpcStream{
		codec: codec,
		request: &rpcRequest{
			codec: codec,
		},
	}

	// Check if sequence is correct
	for i := 0; i < 1000; i++ {
		if err := streamServer.Send(fmt.Sprintf(`{"test":"value %d"}`, i)); err != nil {
			t.Errorf("Unexpected Send error: %s", err)
		}
	}

	for i := 0; i < 1000; i++ {
		var msg string
		if err := streamServer.Recv(&msg); err != nil {
			t.Errorf("Unexpected Recv error: %s", err)
		}
		if msg != fmt.Sprintf(`{"test":"value %d"}`, i) {
			t.Errorf("Unexpected msg: %s", msg)
		}
	}
}

func TestRPCStream_Concurrency(t *testing.T) {
	buffer := new(safeBuffer)
	codec := protoCodec.NewCodec(buffer)
	streamServer := rpcStream{
		codec: codec,
		request: &rpcRequest{
			codec: codec,
		},
	}

	var wg sync.WaitGroup
	// Check if race conditions happen
	for i := 0; i < 10; i++ {
		wg.Add(2)

		go func() {
			for i := 0; i < 50; i++ {
				msg := protoStruct{Payload: "test"}
				<-time.After(time.Duration(rand.Intn(50)) * time.Millisecond)
				if err := streamServer.Send(msg); err != nil {
					t.Errorf("Unexpected Send error: %s", err)
				}
			}
			wg.Done()
		}()

		go func() {
			for i := 0; i < 50; i++ {
				<-time.After(time.Duration(rand.Intn(50)) * time.Millisecond)
				if err := streamServer.Recv(&protoStruct{}); err != nil {
					t.Errorf("Unexpected Recv error: %s", err)
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
}
