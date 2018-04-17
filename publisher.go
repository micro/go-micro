package micro

import (
	"context"

	"github.com/micro/go-micro/client"
)

type publisher struct {
	c     client.Client
	topic string
}

func (p *publisher) Publish(ctx context.Context, msg interface{}, opts ...client.PublishOption) error {
	return p.c.Publish(ctx, p.c.NewMessage(p.topic, msg))
}
