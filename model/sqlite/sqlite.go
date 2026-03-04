// Package sqlite provides a SQLite Database implementation for the model package.
// Uses mattn/go-sqlite3 for broad compatibility.
// Good for development, testing, and single-node production.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"go-micro.dev/v5/model"
)

// Database is a SQLite model.Database implementation.
type Database struct {
	db *sql.DB
}

// New creates a new SQLite database. DSN is the file path (e.g., "data.db" or ":memory:").
func New(dsn string) *Database {
	if dsn == "" {
		dsn = ":memory:"
	}
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		panic(fmt.Sprintf("model/sqlite: failed to open %q: %v", dsn, err))
	}
	// Enable WAL mode for better concurrent read performance
	db.Exec("PRAGMA journal_mode=WAL")
	return &Database{db: db}
}

func (d *Database) Init(opts ...model.Option) error {
	return d.db.Ping()
}

func (d *Database) NewTable(schema *model.Schema) error {
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

	// Create indexes
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

func (d *Database) Create(ctx context.Context, schema *model.Schema, key string, fields map[string]any) error {
	cols, placeholders, values := buildInsert(schema, fields)
	query := fmt.Sprintf("INSERT INTO %q (%s) VALUES (%s)", schema.Table, cols, placeholders)
	_, err := d.db.ExecContext(ctx, query, values...)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") || strings.Contains(err.Error(), "PRIMARY KEY") {
			return model.ErrDuplicateKey
		}
		return fmt.Errorf("model/sqlite: create: %w", err)
	}
	return nil
}

func (d *Database) Read(ctx context.Context, schema *model.Schema, key string) (map[string]any, error) {
	cols := columnList(schema)
	query := fmt.Sprintf("SELECT %s FROM %q WHERE %q = ?", cols, schema.Table, schema.Key)
	row := d.db.QueryRowContext(ctx, query, key)
	return scanRow(schema, row)
}

func (d *Database) Update(ctx context.Context, schema *model.Schema, key string, fields map[string]any) error {
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

func (d *Database) Delete(ctx context.Context, schema *model.Schema, key string) error {
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

func (d *Database) List(ctx context.Context, schema *model.Schema, opts ...model.QueryOption) ([]map[string]any, error) {
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
		return nil, fmt.Errorf("model/sqlite: list: %w", err)
	}
	defer rows.Close()

	return scanRows(schema, rows)
}

func (d *Database) Count(ctx context.Context, schema *model.Schema, opts ...model.QueryOption) (int64, error) {
	q := model.ApplyQueryOptions(opts...)

	query := fmt.Sprintf("SELECT COUNT(*) FROM %q", schema.Table)
	var args []any

	if len(q.Filters) > 0 {
		where, fArgs := buildWhere(q.Filters)
		query += " WHERE " + where
		args = append(args, fArgs...)
	}

	var count int64
	err := d.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("model/sqlite: count: %w", err)
	}
	return count, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) String() string {
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

// newScanPtr returns a pointer suitable for sql.Scan based on the Go type.
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

// derefScanPtr extracts the scanned value and converts to the target Go type.
func derefScanPtr(ptr any, t reflect.Type) any {
	rv := reflect.ValueOf(ptr).Elem()
	if rv.Type().ConvertibleTo(t) {
		return rv.Convert(t).Interface()
	}
	return rv.Interface()
}
