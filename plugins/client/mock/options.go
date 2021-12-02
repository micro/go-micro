package mock

import (
	"go-micro.dev/v4/client"
)

// Response sets the response methods for a service
func Response(service string, response []MockResponse) client.Option {
	return func(o *client.Options) {
		r, ok := responseFromContext(o.Context)
		if !ok {
			r = make(map[string][]MockResponse)
		}
		r[service] = response
		o.Context = newResponseContext(o.Context, r)
	}
}

// Subscriber sets the subscribers service
func Subscriber(topic string, subscriber MockSubscriber) client.Option {
	return func(o *client.Options) {
		r, ok := subscriberFromContext(o.Context)
		if !ok {
			r = make(map[string]MockSubscriber)
		}
		r[topic] = subscriber
		o.Context = newSubscriberContext(o.Context, r)
	}
}
