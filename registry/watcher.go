package registry

type Watcher interface {
	Next() (*Result, error)
	Stop()
}

type Result struct {
	Action  string
	Service *Service
}
