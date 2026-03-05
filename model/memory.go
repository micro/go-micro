package model

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

type memoryModel struct {
	mu      sync.RWMutex
	schemas map[string]*Schema
	types   map[reflect.Type]*Schema
	tables  map[string]map[string]map[string]any // table -> key -> fields
}

func newMemoryModel(opts ...Option) Model {
	return &memoryModel{
		schemas: make(map[string]*Schema),
		types:   make(map[reflect.Type]*Schema),
		tables:  make(map[string]map[string]map[string]any),
	}
}

func (m *memoryModel) Init(opts ...Option) error {
	return nil
}

func (m *memoryModel) Register(v interface{}, opts ...RegisterOption) error {
	schema := BuildSchema(v, opts...)
	t := ResolveType(v)

	m.mu.Lock()
	defer m.mu.Unlock()

	m.schemas[schema.Table] = schema
	m.types[t] = schema
	if _, ok := m.tables[schema.Table]; !ok {
		m.tables[schema.Table] = make(map[string]map[string]any)
	}
	return nil
}

func (m *memoryModel) schema(v interface{}) (*Schema, error) {
	t := ResolveType(v)
	m.mu.RLock()
	s, ok := m.types[t]
	m.mu.RUnlock()
	if !ok {
		return nil, ErrNotRegistered
	}
	return s, nil
}

func (m *memoryModel) Create(ctx context.Context, v interface{}) error {
	schema, err := m.schema(v)
	if err != nil {
		return err
	}
	fields := StructToMap(schema, v)
	key := KeyValue(schema, v)
	if key == "" {
		return fmt.Errorf("model: key field %q not set", schema.Key)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	tbl := m.tables[schema.Table]
	if _, exists := tbl[key]; exists {
		return ErrDuplicateKey
	}
	row := make(map[string]any, len(fields))
	for k, v := range fields {
		row[k] = v
	}
	tbl[key] = row
	return nil
}

func (m *memoryModel) Read(ctx context.Context, key string, v interface{}) error {
	schema, err := m.schema(v)
	if err != nil {
		return err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	tbl := m.tables[schema.Table]
	row, ok := tbl[key]
	if !ok {
		return ErrNotFound
	}
	MapToStruct(schema, row, v)
	return nil
}

func (m *memoryModel) Update(ctx context.Context, v interface{}) error {
	schema, err := m.schema(v)
	if err != nil {
		return err
	}
	fields := StructToMap(schema, v)
	key := KeyValue(schema, v)
	if key == "" {
		return fmt.Errorf("model: key field %q not set", schema.Key)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	tbl := m.tables[schema.Table]
	if _, ok := tbl[key]; !ok {
		return ErrNotFound
	}
	row := make(map[string]any, len(fields))
	for k, v := range fields {
		row[k] = v
	}
	tbl[key] = row
	return nil
}

func (m *memoryModel) Delete(ctx context.Context, key string, v interface{}) error {
	schema, err := m.schema(v)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	tbl := m.tables[schema.Table]
	if _, ok := tbl[key]; !ok {
		return ErrNotFound
	}
	delete(tbl, key)
	return nil
}

func (m *memoryModel) List(ctx context.Context, result interface{}, opts ...QueryOption) error {
	// result must be *[]*T
	rv := reflect.ValueOf(result)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("model: result must be a pointer to a slice")
	}
	sliceVal := rv.Elem()
	elemType := sliceVal.Type().Elem() // *T
	structType := elemType
	if structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}

	m.mu.RLock()
	s, ok := m.types[structType]
	m.mu.RUnlock()
	if !ok {
		return ErrNotRegistered
	}

	q := ApplyQueryOptions(opts...)

	m.mu.RLock()
	tbl := m.tables[s.Table]

	var rows []map[string]any
	for _, row := range tbl {
		if matchFilters(row, q.Filters) {
			cp := make(map[string]any, len(row))
			for k, v := range row {
				cp[k] = v
			}
			rows = append(rows, cp)
		}
	}
	m.mu.RUnlock()

	if q.OrderBy != "" {
		sortRows(rows, q.OrderBy, q.Desc)
	}
	if q.Offset > 0 && uint(len(rows)) > q.Offset {
		rows = rows[q.Offset:]
	} else if q.Offset > 0 {
		rows = nil
	}
	if q.Limit > 0 && uint(len(rows)) > q.Limit {
		rows = rows[:q.Limit]
	}

	results := reflect.MakeSlice(sliceVal.Type(), len(rows), len(rows))
	for i, row := range rows {
		vp := reflect.New(structType)
		MapToStruct(s, row, vp.Interface())
		if elemType.Kind() == reflect.Ptr {
			results.Index(i).Set(vp)
		} else {
			results.Index(i).Set(vp.Elem())
		}
	}
	sliceVal.Set(results)
	return nil
}

func (m *memoryModel) Count(ctx context.Context, v interface{}, opts ...QueryOption) (int64, error) {
	schema, err := m.schema(v)
	if err != nil {
		return 0, err
	}
	q := ApplyQueryOptions(opts...)

	m.mu.RLock()
	defer m.mu.RUnlock()

	tbl := m.tables[schema.Table]
	var count int64
	for _, row := range tbl {
		if matchFilters(row, q.Filters) {
			count++
		}
	}
	return count, nil
}

func (m *memoryModel) Close() error {
	return nil
}

func (m *memoryModel) String() string {
	return "memory"
}

// matchFilters returns true if the row satisfies all filters.
func matchFilters(row map[string]any, filters []Filter) bool {
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

func compareNumeric(a any, op string, b any) bool {
	af, aOk := toFloat64(a)
	bf, bOk := toFloat64(b)
	if !aOk || !bOk {
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
