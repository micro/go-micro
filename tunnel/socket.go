package tunnel

import "github.com/micro/go-micro/transport"

type tunSocket struct{}

func (s *tunSocket) Recv(m *transport.Message) error {
	return nil
}

func (s *tunSocket) Send(m *transport.Message) error {
	return nil
}

func (s *tunSocket) Close() error {
	return nil
}

func (s *tunSocket) Local() string {
	return ""
}

func (s *tunSocket) Remote() string {
	return ""
}
