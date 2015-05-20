package transport

import (
	"bytes"
	"encoding/json"
	"strings"
	"time"

	"github.com/apcera/nats"
)

type NatsTransport struct{}

type NatsTransportClient struct {
	conn   *nats.Conn
	target string
}

type NatsTransportSocket struct {
	m   *nats.Msg
	hdr map[string]string
	buf *bytes.Buffer
}

type NatsTransportServer struct {
	conn *nats.Conn
	name string
	exit chan bool
}

func (n *NatsTransportClient) Send(m *Message) (*Message, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	rsp, err := n.conn.Request(n.target, b, time.Second*10)
	if err != nil {
		return nil, err
	}

	var mr *Message
	if err := json.Unmarshal(rsp.Data, &mr); err != nil {
		return nil, err
	}

	return mr, nil
}

func (n *NatsTransportClient) Close() error {
	n.conn.Close()
	return nil
}

func (n *NatsTransportSocket) Recv() (*Message, error) {
	var m *Message

	if err := json.Unmarshal(n.m.Data, &m); err != nil {
		return nil, err
	}

	return m, nil
}

func (n *NatsTransportSocket) WriteHeader(k string, v string) {
	n.hdr[k] = v
}

func (n *NatsTransportSocket) Write(b []byte) error {
	_, err := n.buf.Write(b)
	return err
}

func (n *NatsTransportServer) Addr() string {
	return "127.0.0.1:4222"
}

func (n *NatsTransportServer) Close() error {
	n.exit <- true
	n.conn.Close()
	return nil
}

func (n *NatsTransportServer) Serve(fn func(Socket)) error {
	s, err := n.conn.QueueSubscribe(n.name, "queue:"+n.name, func(m *nats.Msg) {
		buf := bytes.NewBuffer(nil)
		hdr := make(map[string]string)

		fn(&NatsTransportSocket{
			m:   m,
			hdr: hdr,
			buf: buf,
		})

		mrsp := &Message{
			Header: hdr,
			Body:   buf.Bytes(),
		}

		b, err := json.Marshal(mrsp)
		if err != nil {
			return
		}

		n.conn.Publish(m.Reply, b)
		buf.Reset()
	})
	if err != nil {
		return err
	}

	<-n.exit
	return s.Unsubscribe()
}

func (n *NatsTransport) NewClient(name, addr string) (Client, error) {
	if !strings.HasPrefix(addr, "nats://") {
		addr = nats.DefaultURL
	}

	c, err := nats.Connect(addr)
	if err != nil {
		return nil, err
	}

	return &NatsTransportClient{
		conn:   c,
		target: name,
	}, nil
}

func (n *NatsTransport) NewServer(name, addr string) (Server, error) {
	if !strings.HasPrefix(addr, "nats://") {
		addr = nats.DefaultURL
	}

	c, err := nats.Connect(addr)
	if err != nil {
		return nil, err
	}

	return &NatsTransportServer{
		name: name,
		conn: c,
		exit: make(chan bool, 1),
	}, nil
}

func NewNatsTransport(addrs []string) *NatsTransport {
	return &NatsTransport{}
}
