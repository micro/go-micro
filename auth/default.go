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
func (a *noop) Revoke(sa *ServiceAccount) error {
	return nil
}

// AddRole to the service account
func (a *noop) AddRole(sa *ServiceAccount, r *Role) error {
	return nil
}

// RemoveRole from a service account
func (a *noop) RemoveRole(sa *ServiceAccount, r *Role) error {
	return nil
}
