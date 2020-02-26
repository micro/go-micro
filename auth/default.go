package auth

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	DefaultAuth = NewAuth()
)

// NewAuth returns a new default registry which is memory
func NewAuth(opts ...Option) Auth {
	var options Options
	for _, o := range opts {
		o(&options)
	}
	return &memory{
		accounts: make(map[string]*Account),
		opts:     options,
	}
}

// TODO: replace with https://github.com/nats-io/nkeys
// We'll then register public key in registry to use
type memory struct {
	opts Options
	// accounts
	sync.RWMutex
	accounts map[string]*Account
}

func (n *memory) Init(opts ...Option) error {
	for _, o := range opts {
		o(&n.opts)
	}
	return nil
}

func (n *memory) Options() Options {
	return n.opts
}

func (n *memory) Generate(id string, opts ...GenerateOption) (*Account, error) {
	var options GenerateOptions
	for _, o := range opts {
		o(&options)
	}

	// return a pseudo account
	acc := &Account{
		Id:       id,
		Token:    uuid.New().String(),
		Created:  time.Now(),
		Expiry:   time.Now().Add(time.Hour * 24),
		Metadata: make(map[string]string),
	}

	// set opts
	if len(options.Roles) > 0 {
		acc.Roles = options.Roles
	}
	if options.Metadata != nil {
		acc.Metadata = options.Metadata
	}

	// TODO: don't overwrite
	n.Lock()
	// maybe save by account id?
	n.accounts[acc.Token] = acc
	n.Unlock()

	return acc, nil
}

func (n *memory) Revoke(token string) error {
	n.Lock()
	delete(n.accounts, token)
	n.Unlock()
	return nil
}

func (n *memory) Verify(token string) (*Account, error) {
	n.RLock()
	defer n.RUnlock()
	if acc, ok := n.accounts[token]; ok {
		return acc, nil
	}
	return nil, errors.New("account not found")
}

func (n *memory) String() string {
	return "memory"
}
