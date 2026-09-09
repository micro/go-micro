package grpc

import (
	"testing"

	"go-micro.dev/v6/transport"
)

// func TestGRPCTransportPortRange(t *testing.T) {
// 	tp := NewTransport()

// 	lsn1, err := tp.Listen(":44454-44458")
// 	if err != nil {
// 		t.Errorf("Did not expect an error, got %s", err)
// 	}
// 	expectedPort(t, "44454", lsn1)

// 	lsn2, err := tp.Listen(":44454-44458")
// 	if err != nil {
// 		t.Errorf("Did not expect an error, got %s", err)
// 	}
// 	expectedPort(t, "44455", lsn2)

// 	lsn, err := tp.Listen(":0")
// 	if err != nil {
// 		t.Errorf("Did not expect an error, got %s", err)
// 	}

// 	lsn.Close()
// 	lsn1.Close()
// 	lsn2.Close()
// }

func TestGRPCTransportCommunication(t *testing.T) {
	tr := NewTransport()

	l, err := tr.Listen(":0")
	if err != nil {
		t.Errorf("Unexpected listen err: %v", err)
	}
	defer l.Close()

	fn := func(sock transport.Socket) {
		defer sock.Close()

		for {
			var m transport.Message
			if err := sock.Recv(&m); err != nil {
				return
			}

			if err := sock.Send(&m); err != nil {
				return
			}
		}
	}

	done := make(chan bool)

	go func() {
		if err := l.Accept(fn); err != nil {
			select {
			case <-done:
			default:
				t.Errorf("Unexpected accept err: %v", err)
			}
		}
	}()

	c, err := tr.Dial(l.Addr())
	if err != nil {
		t.Errorf("Unexpected dial err: %v", err)
	}
	defer c.Close()

	m := transport.Message{
		Header: map[string]string{
			"X-Content-Type": "application/json",
		},
		Body: []byte(`{"message": "Hello World"}`),
	}

	if err := c.Send(&m); err != nil {
		t.Errorf("Unexpected send err: %v", err)
	}

	var rm transport.Message

	if err := c.Recv(&rm); err != nil {
		t.Errorf("Unexpected recv err: %v", err)
	}

	if string(rm.Body) != string(m.Body) {
		t.Errorf("Expected %v, got %v", m.Body, rm.Body)
	}

	close(done)
}
