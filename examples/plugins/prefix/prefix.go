// Package prefix is a micro plugin for stripping a path prefix
package prefix

import (
	"net/http"
	"strings"

	"github.com/micro/cli/v2"
	"github.com/micro/micro/v2/plugin"
)

type prefix struct {
	p []string
}

func (p *prefix) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "path_prefix",
			Usage:   "Comma separated list of path prefixes to strip before continuing with request e.g /api,/foo,/bar",
			EnvVars: []string{"PATH_PREFIX"},
		},
	}
}

func (p *prefix) Commands() []*cli.Command {
	return nil
}

func (p *prefix) Handler() plugin.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// strip prefix if we have a match
			for _, prefix := range p.p {
				if strings.HasPrefix(r.URL.Path, prefix) {
					r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
					break
				}
			}
			// serve request
			h.ServeHTTP(w, r)
		})
	}
}

func (p *prefix) Init(ctx *cli.Context) error {
	if prefix := ctx.String("path_prefix"); len(prefix) > 0 {
		p.p = append(p.p, strings.Split(prefix, ",")...)
	}
	return nil
}

func (p *prefix) String() string {
	return "prefix"
}

func NewPlugin(prefixes ...string) plugin.Plugin {
	return &prefix{prefixes}
}
