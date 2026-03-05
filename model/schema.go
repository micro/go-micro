package model

import (
	"fmt"
	"reflect"
	"strings"
)

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

// BuildSchema extracts a Schema from a struct type using reflection.
func BuildSchema(v interface{}, opts ...RegisterOption) *Schema {
	t := reflect.TypeOf(v)
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

	for _, o := range opts {
		o(schema)
	}

	return schema
}

// StructToMap converts a struct pointer to a map of column name → value.
func StructToMap(schema *Schema, v interface{}) map[string]any {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	fields := make(map[string]any, len(schema.Fields))
	for _, f := range schema.Fields {
		fv := rv.FieldByName(f.Name)
		if fv.IsValid() {
			fields[f.Column] = fv.Interface()
		}
	}
	return fields
}

// MapToStruct fills a struct pointer from a map of column name → value.
func MapToStruct(schema *Schema, fields map[string]any, v interface{}) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
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
}

// NewFromSchema creates a new zero-value struct pointer for the given schema's original type.
func NewFromSchema(schema *Schema, rtype reflect.Type) interface{} {
	return reflect.New(rtype).Interface()
}

// KeyValue extracts the key value from a struct using the schema.
func KeyValue(schema *Schema, v interface{}) string {
	fields := StructToMap(schema, v)
	key, ok := fields[schema.Key]
	if !ok {
		return ""
	}
	return fmt.Sprint(key)
}

// ResolveType returns the struct reflect.Type from a value (handles pointers and slices).
func ResolveType(v interface{}) reflect.Type {
	t := reflect.TypeOf(v)
	for t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice {
		t = t.Elem()
	}
	return t
}
