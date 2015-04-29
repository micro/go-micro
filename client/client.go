package client

type RequestFunc func(address string) (err error)

type Client interface {
	NewRequest(service string, f RequestFunc) error
}

var (
	client = NewGRPCClient()
)

func NewRequest(service string, f RequestFunc) error {
	return client.NewRequest(service, f)
}
