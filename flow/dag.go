package flow

type dag interface {
	AddVertex(interface{}) error
	AddEdge(interface{}, interface{}) error
	GetVertex(string) (interface{}, error)
	OrderedDescendants(interface{}) ([]interface{}, error)
	OrderedAncestors(interface{}) ([]interface{}, error)
	Validate() error
}
