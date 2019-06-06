package init

type defaultOptions struct {
	opts *Values
}

type stringKey struct{}

func (d *defaultOptions) Init(opts ...Option) error {
	if d.opts == nil {
		d.opts = new(Values)
	}
	for _, o := range opts {
		if err := d.opts.Option(o); err != nil {
			return err
		}
	}
	return nil
}

func (d *defaultOptions) Options() Options {
	return d
}

func (d *defaultOptions) Value(k interface{}) (interface{}, bool) {
	if d.opts == nil {
		d.opts = new(Values)
	}
	return d.opts.Get(k)
}

func (d *defaultOptions) String() string {
	if d.opts == nil {
		d.opts = new(Values)
	}
	n, ok := d.opts.Get(stringKey{})
	if ok {
		return n.(string)
	}
	return "Values"
}
