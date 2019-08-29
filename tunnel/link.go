package tunnel

import (
	"github.com/google/uuid"
	"github.com/micro/go-micro/transport"
)

func newLink(s transport.Socket) *link {
	return &link{
		Socket: s,
		id:     uuid.New().String(),
	}
}
