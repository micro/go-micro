package vault

import (
	"fmt"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/asim/go-micro/v3/config/source"
)

// Currently a single vault reader
type vault struct {
	secretPath string
	secretName string
	opts       source.Options
	client     *api.Client
}

func (c *vault) Read() (*source.ChangeSet, error) {
	secret, err := c.client.Logical().Read(c.secretPath)
	if err != nil {
		return nil, err
	}

	if secret == nil {
		return nil, fmt.Errorf("source not found: %s", c.secretPath)
	}

	if secret.Data == nil && secret.Warnings != nil {
		return nil, fmt.Errorf("source: %s errors: %v", c.secretPath, secret.Warnings)
	}

	data, err := makeMap(secret.Data, c.secretName)
	if err != nil {
		return nil, fmt.Errorf("error reading data: %v", err)
	}

	b, err := c.opts.Encoder.Encode(data)
	if err != nil {
		return nil, fmt.Errorf("error reading source: %v", err)
	}

	cs := &source.ChangeSet{
		Timestamp: time.Now(),
		Format:    c.opts.Encoder.String(),
		Source:    c.String(),
		Data:      b,
	}
	cs.Checksum = cs.Sum()

	return cs, nil
	//return nil, nil
}

func (c *vault) Write(cs *source.ChangeSet) error {
	return nil
}

func (c *vault) String() string {
	return "vault"
}

func (c *vault) Watch() (source.Watcher, error) {
	w := newWatcher(c.client)

	return w, nil
}

// NewSource creates a new vault source
func NewSource(opts ...source.Option) source.Source {
	options := source.NewOptions(opts...)

	// create the client
	client, _ := api.NewClient(api.DefaultConfig())

	// get and set options
	if address := getAddress(options); address != "" {
		_ = client.SetAddress(address)
	}

	if nameSpace := getNameSpace(options); nameSpace != "" {
		client.SetNamespace(nameSpace)
	}

	if token := getToken(options); token != "" {
		client.SetToken(token)
	}

	path := getResourcePath(options)
	name := getSecretName(options)
	if name == "" {
		name = path
	}

	return &vault{
		opts:       options,
		client:     client,
		secretPath: path,
		secretName: name,
	}
}
