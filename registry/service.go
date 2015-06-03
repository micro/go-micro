package registry

type Service struct {
	Name      string
	Version   string
	Metadata  map[string]string
	Endpoints []*Endpoint
	Nodes     []*Node
}

type Node struct {
	Id       string
	Address  string
	Port     int
	Metadata map[string]string
}

type Endpoint struct {
	Name     string
	Request  *Value
	Response *Value
}

type Value struct {
	Name   string
	Type   string
	Values []*Value
}
