// Package sqlite provides a SQLite model.Model implementation.
// Uses mattn/go-sqlite3 for broad compatibility.
// Good for development, testing, and single-node production.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"sync"

	_ "github.com/mattn/go-sqlite3"

	"go-micro.dev/v5/model"
)

type sqliteModel struct {
	db      *sql.DB
	mu      sync.RWMutex
	schemas map[string]*model.Schema
	types   map[reflect.Type]*model.Schema
}

// New creates a new SQLite model. DSN is the file path (e.g., "data.db" or ":memory:").
func New(dsn string) model.Model {
	if dsn == "" {
		dsn = ":memory:"
	}
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		panic(fmt.Sprintf("model/sqlite: failed to open %q: %v", dsn, err))
	}
	db.Exec("PRAGMA journal_mode=WAL")
	return &sqliteModel{
		db:      db,
		schemas: make(map[string]*model.Schema),
		types:   make(map[reflect.Type]*model.Schema),
	}
}

func (d *sqliteModel) Init(opts ...model.Option) error {
	return d.db.Ping()
}

func (d *sqliteModel) Register(v interface{}, opts ...model.RegisterOption) error {
	schema := model.BuildSchema(v, opts...)
	t := model.ResolveType(v)

	d.mu.Lock()
	d.schemas[schema.Table] = schema
	d.types[t] = schema
	d.mu.Unlock()

	var cols []string
	for _, f := range schema.Fields {
		colType := goTypeToSQLite(f.Type)
		col := fmt.Sprintf("%q %s", f.Column, colType)
		if f.IsKey {
			col += " PRIMARY KEY"
		}
		cols = append(cols, col)
	}

	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %q (%s)", schema.Table, strings.Join(cols, ", "))
	if _, err := d.db.Exec(query); err != nil {
		return fmt.Errorf("model/sqlite: create table: %w", err)
	}

	for _, f := range schema.Fields {
		if f.Index && !f.IsKey {
			idx := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %q ON %q (%q)",
				"idx_"+schema.Table+"_"+f.Column, schema.Table, f.Column)
			if _, err := d.db.Exec(idx); err != nil {
				return fmt.Errorf("model/sqlite: create index: %w", err)
			}
		}
	}

	return nil
}

func (d *sqliteModel) schema(v interface{}) (*model.Schema, error) {
	t := model.ResolveType(v)
	d.mu.RLock()
	s, ok := d.types[t]
	d.mu.RUnlock()
	if !ok {
		return nil, model.ErrNotRegistered
	}
	return s, nil
}

func (d *sqliteModel) Create(ctx context.Context, v interface{}) error {
	schema, err := d.schema(v)
	if err != nil {
		return err
	}
	fields := model.StructToMap(schema, v)
	cols, placeholders, values := buildInsert(schema, fields)
	query := fmt.Sprintf("INSERT INTO %q (%s) VALUES (%s)", schema.Table, cols, placeholders)
	_, err = d.db.ExecContext(ctx, query, values...)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") || strings.Contains(err.Error(), "PRIMARY KEY") {
			return model.ErrDuplicateKey
		}
		return fmt.Errorf("model/sqlite: create: %w", err)
	}
	return nil
}

func (d *sqliteModel) Read(ctx context.Context, key string, v interface{}) error {
	schema, err := d.schema(v)
	if err != nil {
		return err
	}
	cols := columnList(schema)
	query := fmt.Sprintf("SELECT %s FROM %q WHERE %q = ?", cols, schema.Table, schema.Key)
	row := d.db.QueryRowContext(ctx, query, key)
	fields, err := scanRow(schema, row)
	if err != nil {
		return err
	}
	model.MapToStruct(schema, fields, v)
	return nil
}

func (d *sqliteModel) Update(ctx context.Context, v interface{}) error {
	schema, err := d.schema(v)
	if err != nil {
		return err
	}
	fields := model.StructToMap(schema, v)
	key := model.KeyValue(schema, v)
	setClauses, values := buildUpdate(schema, fields)
	values = append(values, key)
	query := fmt.Sprintf("UPDATE %q SET %s WHERE %q = ?", schema.Table, setClauses, schema.Key)
	result, err := d.db.ExecContext(ctx, query, values...)
	if err != nil {
		return fmt.Errorf("model/sqlite: update: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return model.ErrNotFound
	}
	return nil
}

func (d *sqliteModel) Delete(ctx context.Context, key string, v interface{}) error {
	schema, err := d.schema(v)
	if err != nil {
		return err
	}
	query := fmt.Sprintf("DELETE FROM %q WHERE %q = ?", schema.Table, schema.Key)
	result, err := d.db.ExecContext(ctx, query, key)
	if err != nil {
		return fmt.Errorf("model/sqlite: delete: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return model.ErrNotFound
	}
	return nil
}

func (d *sqliteModel) List(ctx context.Context, result interface{}, opts ...model.QueryOption) error {
	// result must be *[]*T
	rv := reflect.ValueOf(result)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("model/sqlite: result must be a pointer to a slice")
	}
	sliceVal := rv.Elem()
	elemType := sliceVal.Type().Elem()
	structType := elemType
	if structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}

	d.mu.RLock()
	schema, ok := d.types[structType]
	d.mu.RUnlock()
	if !ok {
		return model.ErrNotRegistered
	}

	q := model.ApplyQueryOptions(opts...)
	cols := columnList(schema)

	query := fmt.Sprintf("SELECT %s FROM %q", cols, schema.Table)
	var args []any

	if len(q.Filters) > 0 {
		where, fArgs := buildWhere(q.Filters)
		query += " WHERE " + where
		args = append(args, fArgs...)
	}

	if q.OrderBy != "" {
		dir := "ASC"
		if q.Desc {
			dir = "DESC"
		}
		query += fmt.Sprintf(" ORDER BY %q %s", q.OrderBy, dir)
	}

	if q.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", q.Limit)
	}
	if q.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", q.Offset)
	}

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("model/sqlite: list: %w", err)
	}
	defer rows.Close()

	fieldMaps, err := scanRows(schema, rows)
	if err != nil {
		return err
	}

	results := reflect.MakeSlice(sliceVal.Type(), len(fieldMaps), len(fieldMaps))
	for i, fields := range fieldMaps {
		vp := reflect.New(structType)
		model.MapToStruct(schema, fields, vp.Interface())
		if elemType.Kind() == reflect.Ptr {
			results.Index(i).Set(vp)
		} else {
			results.Index(i).Set(vp.Elem())
		}
	}
	sliceVal.Set(results)
	return nil
}

func (d *sqliteModel) Count(ctx context.Context, v interface{}, opts ...model.QueryOption) (int64, error) {
	schema, err := d.schema(v)
	if err != nil {
		return 0, err
	}
	q := model.ApplyQueryOptions(opts...)

	query := fmt.Sprintf("SELECT COUNT(*) FROM %q", schema.Table)
	var args []any

	if len(q.Filters) > 0 {
		where, fArgs := buildWhere(q.Filters)
		query += " WHERE " + where
		args = append(args, fArgs...)
	}

	var count int64
	err = d.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("model/sqlite: count: %w", err)
	}
	return count, nil
}

func (d *sqliteModel) Close() error {
	return d.db.Close()
}

func (d *sqliteModel) String() string {
	return "sqlite"
}

// SQL helpers

func goTypeToSQLite(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "INTEGER"
	case reflect.Float32, reflect.Float64:
		return "REAL"
	case reflect.Bool:
		return "INTEGER"
	default:
		return "TEXT"
	}
}

func buildInsert(schema *model.Schema, fields map[string]any) (string, string, []any) {
	var cols []string
	var placeholders []string
	var values []any
	for _, f := range schema.Fields {
		if v, ok := fields[f.Column]; ok {
			cols = append(cols, fmt.Sprintf("%q", f.Column))
			placeholders = append(placeholders, "?")
			values = append(values, v)
		}
	}
	return strings.Join(cols, ", "), strings.Join(placeholders, ", "), values
}

func buildUpdate(schema *model.Schema, fields map[string]any) (string, []any) {
	var setClauses []string
	var values []any
	for _, f := range schema.Fields {
		if f.IsKey {
			continue
		}
		if v, ok := fields[f.Column]; ok {
			setClauses = append(setClauses, fmt.Sprintf("%q = ?", f.Column))
			values = append(values, v)
		}
	}
	return strings.Join(setClauses, ", "), values
}

func buildWhere(filters []model.Filter) (string, []any) {
	var clauses []string
	var args []any
	for _, f := range filters {
		clauses = append(clauses, fmt.Sprintf("%q %s ?", f.Field, f.Op))
		args = append(args, f.Value)
	}
	return strings.Join(clauses, " AND "), args
}

func columnList(schema *model.Schema) string {
	var cols []string
	for _, f := range schema.Fields {
		cols = append(cols, fmt.Sprintf("%q", f.Column))
	}
	return strings.Join(cols, ", ")
}

func scanRow(schema *model.Schema, row *sql.Row) (map[string]any, error) {
	ptrs := make([]any, len(schema.Fields))
	for i, f := range schema.Fields {
		ptrs[i] = newScanPtr(f.Type)
	}
	if err := row.Scan(ptrs...); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("model/sqlite: scan: %w", err)
	}
	result := make(map[string]any, len(schema.Fields))
	for i, f := range schema.Fields {
		result[f.Column] = derefScanPtr(ptrs[i], f.Type)
	}
	return result, nil
}

func scanRows(schema *model.Schema, rows *sql.Rows) ([]map[string]any, error) {
	var results []map[string]any
	for rows.Next() {
		ptrs := make([]any, len(schema.Fields))
		for i, f := range schema.Fields {
			ptrs[i] = newScanPtr(f.Type)
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("model/sqlite: scan row: %w", err)
		}
		row := make(map[string]any, len(schema.Fields))
		for i, f := range schema.Fields {
			row[f.Column] = derefScanPtr(ptrs[i], f.Type)
		}
		results = append(results, row)
	}
	return results, rows.Err()
}

func newScanPtr(t reflect.Type) any {
	switch t.Kind() {
	case reflect.String:
		return new(string)
	case reflect.Int, reflect.Int64:
		return new(int64)
	case reflect.Int32:
		return new(int32)
	case reflect.Float64:
		return new(float64)
	case reflect.Float32:
		return new(float32)
	case reflect.Bool:
		return new(bool)
	default:
		return new(string)
	}
}

func derefScanPtr(ptr any, t reflect.Type) any {
	rv := reflect.ValueOf(ptr).Elem()
	if rv.Type().ConvertibleTo(t) {
		return rv.Convert(t).Interface()
	}
	return rv.Interface()
}
