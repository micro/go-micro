// Package model provides a typed data model layer with CRUD operations and query support.
// It uses Go generics for type-safe access and supports multiple backends (memory, SQLite, Postgres).
package model

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

var (
	// ErrNotFound is returned when a record doesn't exist.
	ErrNotFound = errors.New("not found")
	// ErrDuplicateKey is returned when a record with the same key already exists.
	ErrDuplicateKey = errors.New("duplicate key")
)

// Database is the backend interface that model implementations must satisfy.
// Each backend (memory, sqlite, postgres) implements this interface.
type Database interface {
	// Init initializes the database connection.
	Init(...Option) error
	// NewTable ensures the table exists for the given schema.
	NewTable(schema *Schema) error
	// Create inserts a new record. Returns ErrDuplicateKey if key exists.
	Create(ctx context.Context, schema *Schema, key string, fields map[string]any) error
	// Read returns a single record by key. Returns ErrNotFound if missing.
	Read(ctx context.Context, schema *Schema, key string) (map[string]any, error)
	// Update modifies an existing record by key. Returns ErrNotFound if missing.
	Update(ctx context.Context, schema *Schema, key string, fields map[string]any) error
	// Delete removes a record by key. Returns ErrNotFound if missing.
	Delete(ctx context.Context, schema *Schema, key string) error
	// List returns all records matching the query options.
	List(ctx context.Context, schema *Schema, opts ...QueryOption) ([]map[string]any, error)
	// Count returns the number of records matching the query options.
	Count(ctx context.Context, schema *Schema, opts ...QueryOption) (int64, error)
	// Close closes the database connection.
	Close() error
	// String returns the implementation name.
	String() string
}

// Schema describes a model's storage layout, derived from struct tags.
type Schema struct {
	// Table name in the database.
	Table string
	// Key is the name of the primary key field.
	Key string
	// Fields maps Go field names to their column metadata.
	Fields []Field
}

// Field describes a single field in the schema.
type Field struct {
	// Name is the Go struct field name.
	Name string
	// Column is the database column name (from json tag or lowercased name).
	Column string
	// Type is the Go reflect type.
	Type reflect.Type
	// IsKey indicates this is the primary key field.
	IsKey bool
	// Index indicates this field should be indexed.
	Index bool
}

// Model provides typed CRUD operations for a specific Go struct type.
type Model[T any] struct {
	db     Database
	schema *Schema
}

// New creates a new Model for the given type T, backed by the provided database.
// T must be a struct with at least one field tagged `model:"key"`.
func New[T any](db Database, opts ...ModelOption) *Model[T] {
	var t T
	schema := buildSchema(reflect.TypeOf(t))

	// Apply model options
	for _, o := range opts {
		o(schema)
	}

	// Ensure table exists
	if err := db.NewTable(schema); err != nil {
		panic(fmt.Sprintf("model: failed to create table %q: %v", schema.Table, err))
	}

	return &Model[T]{
		db:     db,
		schema: schema,
	}
}

// Create inserts a new record.
func (m *Model[T]) Create(ctx context.Context, v *T) error {
	fields := structToMap(m.schema, v)
	key, ok := fields[m.schema.Key]
	if !ok {
		return fmt.Errorf("model: key field %q not set", m.schema.Key)
	}
	return m.db.Create(ctx, m.schema, fmt.Sprint(key), fields)
}

// Read retrieves a record by its primary key.
func (m *Model[T]) Read(ctx context.Context, key string) (*T, error) {
	fields, err := m.db.Read(ctx, m.schema, key)
	if err != nil {
		return nil, err
	}
	v := mapToStruct[T](m.schema, fields)
	return v, nil
}

// Update modifies an existing record.
func (m *Model[T]) Update(ctx context.Context, v *T) error {
	fields := structToMap(m.schema, v)
	key, ok := fields[m.schema.Key]
	if !ok {
		return fmt.Errorf("model: key field %q not set", m.schema.Key)
	}
	return m.db.Update(ctx, m.schema, fmt.Sprint(key), fields)
}

// Delete removes a record by its primary key.
func (m *Model[T]) Delete(ctx context.Context, key string) error {
	return m.db.Delete(ctx, m.schema, key)
}

// List returns records matching the query options.
func (m *Model[T]) List(ctx context.Context, opts ...QueryOption) ([]*T, error) {
	rows, err := m.db.List(ctx, m.schema, opts...)
	if err != nil {
		return nil, err
	}
	results := make([]*T, len(rows))
	for i, row := range rows {
		results[i] = mapToStruct[T](m.schema, row)
	}
	return results, nil
}

// Count returns the number of records matching the query options.
func (m *Model[T]) Count(ctx context.Context, opts ...QueryOption) (int64, error) {
	return m.db.Count(ctx, m.schema, opts...)
}

// Schema returns the model's schema (useful for debugging/introspection).
func (m *Model[T]) Schema() *Schema {
	return m.schema
}

// buildSchema extracts the Schema from a struct type using reflection.
func buildSchema(t reflect.Type) *Schema {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	schema := &Schema{
		Table: strings.ToLower(t.Name()) + "s",
	}

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}

		field := Field{
			Name: f.Name,
			Type: f.Type,
		}

		// Column name: use json tag if present, else lowercase field name
		if tag := f.Tag.Get("json"); tag != "" {
			parts := strings.Split(tag, ",")
			if parts[0] != "" && parts[0] != "-" {
				field.Column = parts[0]
			}
		}
		if field.Column == "" {
			field.Column = strings.ToLower(f.Name)
		}

		// Check model tag
		if tag := f.Tag.Get("model"); tag != "" {
			for _, opt := range strings.Split(tag, ",") {
				switch opt {
				case "key":
					field.IsKey = true
					schema.Key = field.Column
				case "index":
					field.Index = true
				}
			}
		}

		schema.Fields = append(schema.Fields, field)
	}

	if schema.Key == "" {
		// Default to "id" if no key tag found
		for i := range schema.Fields {
			if schema.Fields[i].Column == "id" {
				schema.Fields[i].IsKey = true
				schema.Key = "id"
				break
			}
		}
	}

	return schema
}

// structToMap converts a struct to a map of column name → value.
func structToMap[T any](schema *Schema, v *T) map[string]any {
	rv := reflect.ValueOf(v).Elem()
	fields := make(map[string]any, len(schema.Fields))
	for _, f := range schema.Fields {
		fv := rv.FieldByName(f.Name)
		if fv.IsValid() {
			fields[f.Column] = fv.Interface()
		}
	}
	return fields
}

// mapToStruct converts a map of column name → value back to a struct.
func mapToStruct[T any](schema *Schema, fields map[string]any) *T {
	v := new(T)
	rv := reflect.ValueOf(v).Elem()
	for _, f := range schema.Fields {
		val, ok := fields[f.Column]
		if !ok {
			continue
		}
		fv := rv.FieldByName(f.Name)
		if !fv.IsValid() || !fv.CanSet() {
			continue
		}
		rval := reflect.ValueOf(val)
		if rval.Type().AssignableTo(fv.Type()) {
			fv.Set(rval)
		} else if rval.Type().ConvertibleTo(fv.Type()) {
			fv.Set(rval.Convert(fv.Type()))
		}
	}
	return v
}
