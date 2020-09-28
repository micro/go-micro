// Package secrets is an interface dynamic secret configuration
package secrets

import (
	"github.com/micro/go-micro/v3/config"
)

type Secrets interface {
	config.Config
}
