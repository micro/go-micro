package init

type defaultInit struct {
	opts *Options
}

type stringKey struct{}

func (i *defaultInit) Init(opts ...Option) error {
	if i.opts == nil {
		i.opts = new(Options)
	}
	for _, o := range opts {
		if err := i.opts.SetOption(o); err != nil {
			return err
		}
	}
	return nil
}

func (i *defaultInit) Options() *Options {
	if i.opts == nil {
		i.opts = new(Options)
	}
	return i.opts
}

func (i *defaultInit) Value(k interface{}) (interface{}, bool) {
	if i.opts == nil {
		i.opts = new(Options)
	}
	return i.opts.Value(k)
}

func (i *defaultInit) String() string {
	if i.opts == nil {
		i.opts = new(Options)
	}
	n, ok := i.opts.Value(stringKey{})
	if ok {
		return n.(string)
	}
	return "defaultInit"
}
