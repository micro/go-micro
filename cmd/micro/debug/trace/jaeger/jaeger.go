package jaeger

import (
	"io"

	"github.com/opentracing/opentracing-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
)

// NewTracer returns a new Jaeger tracer based on the current configuration,
// using the given options, and a closer func that can be used to flush buffers
// before shutdown.
func NewTracer(opts ...Option) (opentracing.Tracer, io.Closer, error) {
	options := newOptions(opts...)

	cfg := &jaegercfg.Configuration{}
	if options.FromEnv {
		c, err := jaegercfg.FromEnv()
		if err != nil {
			return nil, nil, err
		}
		cfg = c
	}

	if options.Name != "" {
		cfg.ServiceName = options.Name
	}

	var jOptions []jaegercfg.Option
	if options.Logger != nil {
		jOptions = append(jOptions, jaegercfg.Logger(options.Logger))
	}
	if options.Metrics != nil {
		jOptions = append(jOptions, jaegercfg.Metrics(options.Metrics))
	}

	tracer, closer, err := cfg.NewTracer(jOptions...)
	if err != nil {
		return nil, nil, err
	}

	if options.GlobalTracer {
		opentracing.SetGlobalTracer(tracer)
	}

	return tracer, closer, nil
}
