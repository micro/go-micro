package micro

import (
	"github.com/micro/go-micro/client"
	"golang.org/x/net/context"
)

type publisher struct {
	c     client.Client
	topic string
}

func (p *publisher) Publish(ctx context.Context, msg interface{}, opts ...client.PublishOption) error {
	return p.c.Publish(ctx, p.c.NewPublication(p.topic, msg))
}
