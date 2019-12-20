package client

type LogOptions struct {
	AdditionalParams map[string]string
}

type LogOption func(*LogOptions)

func AdditionalParams(p map[string]string) LogOption {
	return func(l *LogOptions) {
		l.AdditionalParams = p
	}
}
