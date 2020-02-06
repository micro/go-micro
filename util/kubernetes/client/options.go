package client

type DeploymentOptions struct {
	BaseImage string
}

type LogOptions struct {
	Params map[string]string
}

type WatchOptions struct {
	Params map[string]string
}

type LogOption func(*LogOptions)
type WatchOption func(*WatchOptions)
type DeploymentOption func(*DeploymentOptions)

// LogParams provides additional params for logs
func LogParams(p map[string]string) LogOption {
	return func(l *LogOptions) {
		l.Params = p
	}
}

// WatchParams used for watch params
func WatchParams(p map[string]string) WatchOption {
	return func(w *WatchOptions) {
		w.Params = p
	}
}

// WithBaseImage sets the base image for the deployment
func WithBaseImage(img string) DeploymentOption {
	return func(d *DeploymentOptions) {
		d.BaseImage = img
	}
}

// NewDeploymentOptions returns an initialized DeploymentOptions
func NewDeploymentOptions(opts []DeploymentOption) DeploymentOptions {
	var options DeploymentOptions
	for _, o := range opts {
		o(&options)
	}

	if options.BaseImage == "" {
		options.BaseImage = DefaultImage
	}

	return options
}
