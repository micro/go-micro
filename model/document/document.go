// Package document provides a document-oriented crud interface
package document

type Document interface {
	// Read values
	Read(...ReadOption) ([]Record, error)
	// Write a value
	Write(id string, v interface{}) error
	// Update a record
	Update(id string, v interface{}) error
	// Delete a record
	Delete(id string) error
}

type Record interface {
	// Scan the value into an interface
	Scan(...interface{}) error
}

type ReadOption func(o *ReadOptions)

type ReadOptions struct {
	// Id to read
	Id string
	// Limit number of returns records
	Limit int
	// Offset
	Offset int
}
