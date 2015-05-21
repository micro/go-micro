package transport

import (
	"encoding/json"
	"errors"
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
	conn *nats.Conn
	m    *nats.Msg
}

type NatsTransportListener struct {
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

func (n *NatsTransportSocket) Recv(m *Message) error {
	if m == nil {
		return errors.New("message passed in is nil")
	}

	if err := json.Unmarshal(n.m.Data, &m); err != nil {
		return err
	}
	return nil
}

func (n *NatsTransportSocket) Send(m *Message) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return n.conn.Publish(n.m.Reply, b)
}

func (n *NatsTransportSocket) Close() error {
	return nil
}

func (n *NatsTransportListener) Addr() string {
	return n.addr
}

func (n *NatsTransportListener) Close() error {
	n.exit <- true
	n.conn.Close()
	return nil
}

func (n *NatsTransportListener) Accept(fn func(Socket)) error {
	s, err := n.conn.Subscribe(n.addr, func(m *nats.Msg) {
		fn(&NatsTransportSocket{
			conn: n.conn,
			m:    m,
		})
	})
	if err != nil {
		return err
	}

	<-n.exit
	return s.Unsubscribe()
}

func (n *NatsTransport) Dial(addr string) (Client, error) {
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

func (n *NatsTransport) Listen(addr string) (Listener, error) {
	cAddr := nats.DefaultURL

	if len(n.addrs) > 0 && strings.HasPrefix(n.addrs[0], "nats://") {
		cAddr = n.addrs[0]
	}

	c, err := nats.Connect(cAddr)
	if err != nil {
		return nil, err
	}

	return &NatsTransportListener{
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
