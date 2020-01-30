package service

import (
	"github.com/micro/go-micro/auth"
)

// Auth is the service implementation of the Auth interface
type Auth struct {
	options auth.Options
}

// Generate a new auth ServiceAccount
func (a *Auth) Generate(sa *auth.ServiceAccount) (*auth.ServiceAccount, error) {
	return nil, nil
}

// Revoke an authorization ServiceAccount
func (a *Auth) Revoke(sa *auth.ServiceAccount) error {
	return nil
}

// AddRole to the service account
func (a *Auth) AddRole(sa *auth.ServiceAccount, r *auth.Role) error {
	return nil
}

// RemoveRole from a service account
func (a *Auth) RemoveRole(sa *auth.ServiceAccount, r *auth.Role) error {
	return nil
}

// NewAuth returns a new instance of the Auth service
func NewAuth(opts ...auth.Option) auth.Auth {
	options := auth.Options{}

	for _, o := range opts {
		o(&options)
	}

	return &Auth{options}
}
