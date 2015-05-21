package transport

import (
	"bytes"
	"encoding/json"
	"strings"
	"time"

	"github.com/apcera/nats"
)

type NatsTransport struct {
	addrs []string
}

type NatsTransportClient struct {
	conn *nats.Conn
	addr string
}

type NatsTransportSocket struct {
	m   *nats.Msg
	hdr map[string]string
	buf *bytes.Buffer
}

type NatsTransportServer struct {
	conn *nats.Conn
	addr string
	exit chan bool
}

func (n *NatsTransportClient) Send(m *Message) (*Message, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	rsp, err := n.conn.Request(n.addr, b, time.Second*10)
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
	return n.addr
}

func (n *NatsTransportServer) Close() error {
	n.exit <- true
	n.conn.Close()
	return nil
}

func (n *NatsTransportServer) Serve(fn func(Socket)) error {
	s, err := n.conn.Subscribe(n.addr, func(m *nats.Msg) {
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

func (n *NatsTransport) NewClient(addr string) (Client, error) {
	cAddr := nats.DefaultURL

	if len(n.addrs) > 0 && strings.HasPrefix(n.addrs[0], "nats://") {
		cAddr = n.addrs[0]
	}

	c, err := nats.Connect(cAddr)
	if err != nil {
		return nil, err
	}

	return &NatsTransportClient{
		conn: c,
		addr: addr,
	}, nil
}

func (n *NatsTransport) NewServer(addr string) (Server, error) {
	cAddr := nats.DefaultURL

	if len(n.addrs) > 0 && strings.HasPrefix(n.addrs[0], "nats://") {
		cAddr = n.addrs[0]
	}

	c, err := nats.Connect(cAddr)
	if err != nil {
		return nil, err
	}

	return &NatsTransportServer{
		addr: nats.NewInbox(),
		conn: c,
		exit: make(chan bool, 1),
	}, nil
}

func NewNatsTransport(addrs []string) *NatsTransport {
	return &NatsTransport{
		addrs: addrs,
	}
}
