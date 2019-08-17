package micro

import (
	"context"

	"github.com/micro/go-micro/client"
)

type publisher struct {
	c           client.Client
	topic       string
	contentType string
}

func (p *publisher) Publish(ctx context.Context, msg interface{}, opts ...client.PublishOption) error {
	var mopts = make([]client.MessageOption, 0)
	if len(p.contentType) > 0 {
		mopts = append(mopts, func(options *client.MessageOptions) {
			options.ContentType = p.contentType
		})
	}
	return p.c.Publish(ctx, p.c.NewMessage(p.topic, msg, mopts...), opts...)
}
