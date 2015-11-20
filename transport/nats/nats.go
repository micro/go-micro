package nats

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/apcera/nats"
	"github.com/micro/go-micro/transport"
)

type ntport struct {
	addrs []string
}

type ntportClient struct {
	conn *nats.Conn
	addr string
	id   string
	sub  *nats.Subscription
}

type ntportSocket struct {
	conn *nats.Conn
	m    *nats.Msg
}

type ntportListener struct {
	conn *nats.Conn
	addr string
	exit chan bool
}

func (n *ntportClient) Send(m *transport.Message) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}

	return n.conn.PublishRequest(n.addr, n.id, b)
}

func (n *ntportClient) Recv(m *transport.Message) error {
	rsp, err := n.sub.NextMsg(time.Second * 10)
	if err != nil {
		return err
	}

	var mr *transport.Message
	if err := json.Unmarshal(rsp.Data, &mr); err != nil {
		return err
	}

	*m = *mr
	return nil
}

func (n *ntportClient) Close() error {
	n.sub.Unsubscribe()
	n.conn.Close()
	return nil
}

func (n *ntportSocket) Recv(m *transport.Message) error {
	if m == nil {
		return errors.New("message passed in is nil")
	}

	if err := json.Unmarshal(n.m.Data, &m); err != nil {
		return err
	}
	return nil
}

func (n *ntportSocket) Send(m *transport.Message) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return n.conn.Publish(n.m.Reply, b)
}

func (n *ntportSocket) Close() error {
	return nil
}

func (n *ntportListener) Addr() string {
	return n.addr
}

func (n *ntportListener) Close() error {
	n.exit <- true
	n.conn.Close()
	return nil
}

func (n *ntportListener) Accept(fn func(transport.Socket)) error {
	s, err := n.conn.Subscribe(n.addr, func(m *nats.Msg) {
		fn(&ntportSocket{
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

func (n *ntport) Dial(addr string, opts ...transport.DialOption) (transport.Client, error) {
	cAddr := nats.DefaultURL

	if len(n.addrs) > 0 && strings.HasPrefix(n.addrs[0], "nats://") {
		cAddr = n.addrs[0]
	}

	c, err := nats.Connect(cAddr)
	if err != nil {
		return nil, err
	}

	id := nats.NewInbox()
	sub, err := c.SubscribeSync(id)
	if err != nil {
		return nil, err
	}

	return &ntportClient{
		conn: c,
		addr: addr,
		id:   id,
		sub:  sub,
	}, nil
}

func (n *ntport) Listen(addr string) (transport.Listener, error) {
	cAddr := nats.DefaultURL

	if len(n.addrs) > 0 && strings.HasPrefix(n.addrs[0], "nats://") {
		cAddr = n.addrs[0]
	}

	c, err := nats.Connect(cAddr)
	if err != nil {
		return nil, err
	}

	return &ntportListener{
		addr: nats.NewInbox(),
		conn: c,
		exit: make(chan bool, 1),
	}, nil
}

func NewTransport(addrs []string, opt ...transport.Option) transport.Transport {
	return &ntport{
		addrs: addrs,
	}
}
