// Package memory provides an in-memory Database implementation for the model package.
// Useful for testing and development. Data does not persist across restarts.
package memory

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"go-micro.dev/v5/model"
)

// Database is an in-memory model.Database implementation.
type Database struct {
	mu     sync.RWMutex
	tables map[string]*table
}

type table struct {
	rows map[string]map[string]any
}

// New creates a new in-memory database.
func New(opts ...model.Option) *Database {
	return &Database{
		tables: make(map[string]*table),
	}
}

func (d *Database) Init(opts ...model.Option) error {
	return nil
}

func (d *Database) NewTable(schema *model.Schema) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.tables[schema.Table]; !ok {
		d.tables[schema.Table] = &table{rows: make(map[string]map[string]any)}
	}
	return nil
}

func (d *Database) Create(ctx context.Context, schema *model.Schema, key string, fields map[string]any) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	t := d.tables[schema.Table]
	if _, exists := t.rows[key]; exists {
		return model.ErrDuplicateKey
	}
	// Copy the map to avoid external mutation
	row := make(map[string]any, len(fields))
	for k, v := range fields {
		row[k] = v
	}
	t.rows[key] = row
	return nil
}

func (d *Database) Read(ctx context.Context, schema *model.Schema, key string) (map[string]any, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	t := d.tables[schema.Table]
	row, ok := t.rows[key]
	if !ok {
		return nil, model.ErrNotFound
	}
	// Return a copy
	result := make(map[string]any, len(row))
	for k, v := range row {
		result[k] = v
	}
	return result, nil
}

func (d *Database) Update(ctx context.Context, schema *model.Schema, key string, fields map[string]any) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	t := d.tables[schema.Table]
	if _, ok := t.rows[key]; !ok {
		return model.ErrNotFound
	}
	row := make(map[string]any, len(fields))
	for k, v := range fields {
		row[k] = v
	}
	t.rows[key] = row
	return nil
}

func (d *Database) Delete(ctx context.Context, schema *model.Schema, key string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	t := d.tables[schema.Table]
	if _, ok := t.rows[key]; !ok {
		return model.ErrNotFound
	}
	delete(t.rows, key)
	return nil
}

func (d *Database) List(ctx context.Context, schema *model.Schema, opts ...model.QueryOption) ([]map[string]any, error) {
	q := model.ApplyQueryOptions(opts...)

	d.mu.RLock()
	defer d.mu.RUnlock()
	t := d.tables[schema.Table]

	var results []map[string]any
	for _, row := range t.rows {
		if matchFilters(row, q.Filters) {
			// Copy row
			cp := make(map[string]any, len(row))
			for k, v := range row {
				cp[k] = v
			}
			results = append(results, cp)
		}
	}

	// Sort if OrderBy is set
	if q.OrderBy != "" {
		sortRows(results, q.OrderBy, q.Desc)
	}

	// Apply offset
	if q.Offset > 0 && uint(len(results)) > q.Offset {
		results = results[q.Offset:]
	} else if q.Offset > 0 {
		results = nil
	}

	// Apply limit
	if q.Limit > 0 && uint(len(results)) > q.Limit {
		results = results[:q.Limit]
	}

	return results, nil
}

func (d *Database) Count(ctx context.Context, schema *model.Schema, opts ...model.QueryOption) (int64, error) {
	q := model.ApplyQueryOptions(opts...)

	d.mu.RLock()
	defer d.mu.RUnlock()
	t := d.tables[schema.Table]

	var count int64
	for _, row := range t.rows {
		if matchFilters(row, q.Filters) {
			count++
		}
	}
	return count, nil
}

func (d *Database) Close() error {
	return nil
}

func (d *Database) String() string {
	return "memory"
}

// matchFilters returns true if the row satisfies all filters.
func matchFilters(row map[string]any, filters []model.Filter) bool {
	for _, f := range filters {
		val, ok := row[f.Field]
		if !ok {
			return false
		}
		if !compareValues(val, f.Op, f.Value) {
			return false
		}
	}
	return true
}

// compareValues compares two values with the given operator.
func compareValues(a any, op string, b any) bool {
	switch op {
	case "=":
		return fmt.Sprint(a) == fmt.Sprint(b)
	case "!=":
		return fmt.Sprint(a) != fmt.Sprint(b)
	case "LIKE":
		// Simple LIKE: supports % wildcard at start/end
		pattern := fmt.Sprint(b)
		val := fmt.Sprint(a)
		if strings.HasPrefix(pattern, "%") && strings.HasSuffix(pattern, "%") {
			return strings.Contains(val, pattern[1:len(pattern)-1])
		}
		if strings.HasPrefix(pattern, "%") {
			return strings.HasSuffix(val, pattern[1:])
		}
		if strings.HasSuffix(pattern, "%") {
			return strings.HasPrefix(val, pattern[:len(pattern)-1])
		}
		return val == pattern
	case "<", ">", "<=", ">=":
		return compareNumeric(a, op, b)
	default:
		return false
	}
}

// compareNumeric attempts numeric comparison.
func compareNumeric(a any, op string, b any) bool {
	af, aOk := toFloat64(a)
	bf, bOk := toFloat64(b)
	if !aOk || !bOk {
		// Fall back to string comparison
		as, bs := fmt.Sprint(a), fmt.Sprint(b)
		switch op {
		case "<":
			return as < bs
		case ">":
			return as > bs
		case "<=":
			return as <= bs
		case ">=":
			return as >= bs
		}
		return false
	}
	switch op {
	case "<":
		return af < bf
	case ">":
		return af > bf
	case "<=":
		return af <= bf
	case ">=":
		return af >= bf
	}
	return false
}

func toFloat64(v any) (float64, bool) {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(rv.Uint()), true
	case reflect.Float32, reflect.Float64:
		return rv.Float(), true
	default:
		return 0, false
	}
}

// sortRows sorts rows by a field. Simple insertion sort for small datasets.
func sortRows(rows []map[string]any, field string, desc bool) {
	for i := 1; i < len(rows); i++ {
		for j := i; j > 0; j-- {
			a := fmt.Sprint(rows[j-1][field])
			b := fmt.Sprint(rows[j][field])
			shouldSwap := a > b
			if desc {
				shouldSwap = a < b
			}
			if shouldSwap {
				rows[j-1], rows[j] = rows[j], rows[j-1]
			}
		}
	}
}
