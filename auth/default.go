package auth

var (
	DefaultAuth Auth = new(noop)
)

type noop struct {
	options Options
}

// Generate a new auth ServiceAccount
func (a *noop) Generate(sa *ServiceAccount) (*ServiceAccount, error) {
	return nil, nil
}

// Revoke an authorization ServiceAccount
func (a *noop) Revoke(token string) error {
	return nil
}

// Validate a service account token
func (a *noop) Validate(token string) (*ServiceAccount, error) {
	return nil, nil
}
