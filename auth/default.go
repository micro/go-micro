package auth

var (
	DefaultAuth Auth = new(noop)
)

type noop struct {
	options Options
}

// String name of implementation
func (a *noop) String() string {
	return "noop"
}

// Init the svc
func (a *noop) Init(...Option) error {
	return nil
}

// Options set in init
func (a *noop) Options() Options {
	return a.options
}

// Generate a new auth Account
func (a *noop) Generate(id string, ops ...GenerateOption) (*Account, error) {
	return nil, nil
}

// Revoke an authorization Account
func (a *noop) Revoke(token string) error {
	return nil
}

// Validate a  account token
func (a *noop) Validate(token string) (*Account, error) {
	return nil, nil
}
