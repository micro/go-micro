package apex

import (
	"context"

	apexLog "github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/apex/log/handlers/json"
	"github.com/apex/log/handlers/text"
	log "github.com/micro/go-micro/v2/logger"
)

type handlerKey struct{}
type levelKey struct{}

// Options is used when applying custom options
type Options struct {
	log.Options
}

// WithLevel allows to set the level for Log Output
func WithLevel(level log.Level) log.Option {
	return setOption(levelKey{}, level)
}

// WithHandler allows to set a customHandler for Log Output
func WithHandler(handler apexLog.Handler) log.Option {
	return setOption(handlerKey{}, handler)
}

// WithTextHandler sets the Text Handler for Log Output
func WithTextHandler() log.Option {
	return WithHandler(text.Default)
}

// WithJSONHandler sets the JSON Handler for Log Output
func WithJSONHandler() log.Option {
	return WithHandler(json.Default)
}

// WithCLIHandler sets the CLI Handler for Log Output
func WithCLIHandler() log.Option {
	return WithHandler(cli.Default)
}

func setOption(k, v interface{}) log.Option {
	return func(o *log.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, k, v)
	}
}
