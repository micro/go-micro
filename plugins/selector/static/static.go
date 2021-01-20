// Package static is a selector which always returns the name specified with a port-number appended.
// AN optional domain-name will also be added.
package static

import (
	"fmt"
	"os"

	"github.com/asim/go-micro/v3/selector"
	"github.com/asim/go-micro/v3/cmd"
	"github.com/asim/go-micro/v3/registry"
)

const (
	ENV_STATIC_SELECTOR_DOMAIN_NAME = "STATIC_SELECTOR_DOMAIN_NAME"
	ENV_STATIC_SELECTOR_PORT_NUMBER = "STATIC_SELECTOR_PORT_NUMBER"
	DEFAULT_PORT_NUMBER             = "8080"
)

type staticSelector struct {
	addressSuffix string
	envDomainName string
	envPortNumber string
}

func init() {
	cmd.DefaultSelectors["static"] = NewSelector
}

func (s *staticSelector) Init(opts ...selector.Option) error {
	return nil
}

func (s *staticSelector) Options() selector.Options {
	return selector.Options{}
}

func (s *staticSelector) Select(service string, opts ...selector.SelectOption) (selector.Next, error) {
	node := &registry.Node{
		Id:      service,
		Address: fmt.Sprintf("%v%v", service, s.addressSuffix),
	}

	return func() (*registry.Node, error) {
		return node, nil
	}, nil
}

func (s *staticSelector) Mark(service string, node *registry.Node, err error) {
	return
}

func (s *staticSelector) Reset(service string) {
	return
}

func (s *staticSelector) Close() error {
	return nil
}

func (s *staticSelector) String() string {
	return "static"
}

func NewSelector(opts ...selector.Option) selector.Selector {

	// Build a new
	s := &staticSelector{
		addressSuffix: "",
		envDomainName: os.Getenv(ENV_STATIC_SELECTOR_DOMAIN_NAME),
		envPortNumber: os.Getenv(ENV_STATIC_SELECTOR_PORT_NUMBER),
	}

	// Add the dns domain-name (if one was specified by an env-var):
	if s.envDomainName != "" {
		s.addressSuffix += fmt.Sprintf(".%v", s.envDomainName)
	}

	// Either add the default port-number, or override with one specified by an env-var:
	if s.envPortNumber == "" {
		s.addressSuffix += fmt.Sprintf(":%v", DEFAULT_PORT_NUMBER)
	} else {
		s.addressSuffix += fmt.Sprintf(":%v", s.envPortNumber)
	}

	return s
}
