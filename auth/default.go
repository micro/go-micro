package auth

import (
	"encoding/base32"
	"sync"
	"time"
)

var (
	DefaultAuth = NewAuth()
)

func genAccount(id string) *Account {
	// return a pseudo account
	return &Account{
		Id:       id,
		Token:    base32.StdEncoding.EncodeToString([]byte(id)),
		Created:  time.Now(),
		Expiry:   time.Now().Add(time.Hour * 24),
		Metadata: make(map[string]string),
	}
}

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
	acc := genAccount(id)

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

	if len(token) == 0 {
		// pseudo account?
		return genAccount(""), nil
	}

	// try get the local account if it exists
	if acc, ok := n.accounts[token]; ok {
		return acc, nil
	}

	// decode the token otherwise
	b, err := base32.StdEncoding.DecodeString(token)
	if err != nil {
		return genAccount(""), nil
	}

	// return a pseudo account based on token/id
	return &Account{
		Id:       string(b),
		Token:    token,
		Created:  time.Now(),
		Expiry:   time.Now().Add(time.Hour * 24),
		Metadata: make(map[string]string),
	}, nil
}

func (n *memory) String() string {
	return "memory"
}
