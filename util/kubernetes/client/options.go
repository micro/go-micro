package client

type CreateOptions struct {
	Namespace string
}

type GetOptions struct {
	Namespace string
	Labels    map[string]string
}
type UpdateOptions struct {
	Namespace string
}
type DeleteOptions struct {
	Namespace string
}
type ListOptions struct {
	Namespace string
}

type LogOptions struct {
	Namespace string
	Params    map[string]string
}

type WatchOptions struct {
	Namespace string
	Params    map[string]string
}

type CreateOption func(*CreateOptions)
type GetOption func(*GetOptions)
type UpdateOption func(*UpdateOptions)
type DeleteOption func(*DeleteOptions)
type ListOption func(*ListOptions)
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

// CreateNamespace sets the namespace for creating a resource
func CreateNamespace(ns string) CreateOption {
	return func(o *CreateOptions) {
		o.Namespace = SerializeResourceName(ns)
	}
}

// GetNamespace sets the namespace for getting a resource
func GetNamespace(ns string) GetOption {
	return func(o *GetOptions) {
		o.Namespace = SerializeResourceName(ns)
	}
}

// GetLabels sets the labels for when getting a resource
func GetLabels(ls map[string]string) GetOption {
	return func(o *GetOptions) {
		o.Labels = ls
	}
}

// UpdateNamespace sets the namespace for updating a resource
func UpdateNamespace(ns string) UpdateOption {
	return func(o *UpdateOptions) {
		o.Namespace = SerializeResourceName(ns)
	}
}

// DeleteNamespace sets the namespace for deleting a resource
func DeleteNamespace(ns string) DeleteOption {
	return func(o *DeleteOptions) {
		o.Namespace = SerializeResourceName(ns)
	}
}

// ListNamespace sets the namespace for listing resources
func ListNamespace(ns string) ListOption {
	return func(o *ListOptions) {
		o.Namespace = SerializeResourceName(ns)
	}
}

// LogNamespace sets the namespace for logging a resource
func LogNamespace(ns string) LogOption {
	return func(o *LogOptions) {
		o.Namespace = SerializeResourceName(ns)
	}
}

// WatchNamespace sets the namespace for watching a resource
func WatchNamespace(ns string) WatchOption {
	return func(o *WatchOptions) {
		o.Namespace = SerializeResourceName(ns)
	}
}
