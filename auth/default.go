package auth

var (
	DefaultAuth = NewAuth()
)

// NewAuth returns a new default registry which is noop
func NewAuth(opts ...Option) Auth {
	var options Options
	for _, o := range opts {
		o(&options)
	}
	return &noop{
		opts: options,
	}
}

type noop struct {
	opts Options
}

func (n *noop) Init(opts ...Option) error {
	for _, o := range opts {
		o(&n.opts)
	}
	return nil
}

func (n *noop) Options() Options {
	return n.opts
}

func (n *noop) Generate(id string, opts ...GenerateOption) (*Account, error) {
	return nil, nil
}

func (n *noop) Revoke(token string) error {
	return nil
}

func (n *noop) Verify(token string) (*Account, error) {
	return nil, nil
}

func (n *noop) String() string {
	return "noop"
}
