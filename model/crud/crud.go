// Package crud provides a crud interface
package crud

type CRUD interface {
	// Read values
	Read(...ReadOption) ([]Record, error)
	// Write a value
	Write(id string, v interface{}, ...WriteOption) error
	// Update a record
	Update(id string, v interface{}, ...UpdateOption) error
	// Delete a record
	Delete(id string, ...DeleteOption) error
}

type Record interface {
	// Value
	Value() []byte
	// Scan the value into an interface
	Scan(v interface{}) error
}

type ReadOption struct {
	// Id to read
	Id string
	// Limit number of returns records
	Limit int
	// Offset
	Offset int
}
