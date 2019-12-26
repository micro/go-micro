package client

type LogOptions struct {
	Params map[string]string
}

type LogOption func(*LogOptions)

// LogParams provides additional params for logs
func LogParams(p map[string]string) LogOption {
	return func(l *LogOptions) {
		l.Params = p
	}
}
