package client

type LogOptions struct {
	Params map[string]string
}

type WatchOptions struct {
	Params map[string]string
}

type LogOption func(*LogOptions)
type WatchOption func(*WatchOptions)

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
