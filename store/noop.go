package store

type noopStore struct{}

func (n *noopStore) Init(opts ...Option) error {
	return nil
}

func (n *noopStore) Options() Options {
	return Options{}
}

func (n *noopStore) String() string {
	return "memory"
}

func (n *noopStore) Read(key string, opts ...ReadOption) ([]*Record, error) {
	return nil, nil
}

func (n *noopStore) Write(r *Record, opts ...WriteOption) error {
	return nil
}

func (n *noopStore) Delete(key string, opts ...DeleteOption) error {
	return nil
}

func (n *noopStore) List(opts ...ListOption) ([]string, error) {
	return nil, nil
}
