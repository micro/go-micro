package auth

var (
	DefaultAuth = NewAuth()
)

// NewAuth returns a new default registry which is noop
func NewAuth(opts ...Option) Auth {
	return noop{}
}

type noop struct{}

func (noop) Init(opts ...Option) error {
	return nil
}

func (noop) Options() Options {
	return Options{}
}

func (noop) Generate(id string, opts ...GenerateOption) (*Account, error) {
	return nil, nil
}

func (noop) Revoke(token string) error {
	return nil
}

func (noop) Validate(token string) (*Account, error) {
	return nil, nil
}

func (noop) String() string {
	return "noop"
}
