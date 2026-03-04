// Package postgres provides a PostgreSQL Database implementation for the model package.
// Uses lib/pq driver. Best for production deployments with rich query support.
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	_ "github.com/lib/pq"

	"go-micro.dev/v5/model"
)

// Database is a PostgreSQL model.Database implementation.
type Database struct {
	db *sql.DB
}

// New creates a new Postgres database. DSN is a connection string
// (e.g., "postgres://user:pass@localhost/dbname?sslmode=disable").
func New(dsn string) *Database {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		panic(fmt.Sprintf("model/postgres: failed to open: %v", err))
	}
	return &Database{db: db}
}

func (d *Database) Init(opts ...model.Option) error {
	return d.db.Ping()
}

func (d *Database) NewTable(schema *model.Schema) error {
	var cols []string
	for _, f := range schema.Fields {
		colType := goTypeToPostgres(f.Type)
		col := fmt.Sprintf("%s %s", quoteIdent(f.Column), colType)
		if f.IsKey {
			col += " PRIMARY KEY"
		}
		cols = append(cols, col)
	}

	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", quoteIdent(schema.Table), strings.Join(cols, ", "))
	if _, err := d.db.Exec(query); err != nil {
		return fmt.Errorf("model/postgres: create table: %w", err)
	}

	// Create indexes
	for _, f := range schema.Fields {
		if f.Index && !f.IsKey {
			idx := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (%s)",
				quoteIdent("idx_"+schema.Table+"_"+f.Column),
				quoteIdent(schema.Table),
				quoteIdent(f.Column))
			if _, err := d.db.Exec(idx); err != nil {
				return fmt.Errorf("model/postgres: create index: %w", err)
			}
		}
	}

	return nil
}

func (d *Database) Create(ctx context.Context, schema *model.Schema, key string, fields map[string]any) error {
	cols, placeholders, values := buildInsert(schema, fields)
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", quoteIdent(schema.Table), cols, placeholders)
	_, err := d.db.ExecContext(ctx, query, values...)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			return model.ErrDuplicateKey
		}
		return fmt.Errorf("model/postgres: create: %w", err)
	}
	return nil
}

func (d *Database) Read(ctx context.Context, schema *model.Schema, key string) (map[string]any, error) {
	cols := columnList(schema)
	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s = $1", cols, quoteIdent(schema.Table), quoteIdent(schema.Key))
	row := d.db.QueryRowContext(ctx, query, key)
	return scanRow(schema, row)
}

func (d *Database) Update(ctx context.Context, schema *model.Schema, key string, fields map[string]any) error {
	setClauses, values := buildUpdate(schema, fields)
	values = append(values, key)
	paramIdx := len(values)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = $%d",
		quoteIdent(schema.Table), setClauses, quoteIdent(schema.Key), paramIdx)
	result, err := d.db.ExecContext(ctx, query, values...)
	if err != nil {
		return fmt.Errorf("model/postgres: update: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return model.ErrNotFound
	}
	return nil
}

func (d *Database) Delete(ctx context.Context, schema *model.Schema, key string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE %s = $1", quoteIdent(schema.Table), quoteIdent(schema.Key))
	result, err := d.db.ExecContext(ctx, query, key)
	if err != nil {
		return fmt.Errorf("model/postgres: delete: %w", err)
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

	query := fmt.Sprintf("SELECT %s FROM %s", cols, quoteIdent(schema.Table))
	var args []any
	paramN := 1

	if len(q.Filters) > 0 {
		where, fArgs, nextParam := buildWhere(q.Filters, paramN)
		query += " WHERE " + where
		args = append(args, fArgs...)
		paramN = nextParam
	}

	if q.OrderBy != "" {
		dir := "ASC"
		if q.Desc {
			dir = "DESC"
		}
		query += fmt.Sprintf(" ORDER BY %s %s", quoteIdent(q.OrderBy), dir)
	}

	if q.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", q.Limit)
	}
	if q.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", q.Offset)
	}

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("model/postgres: list: %w", err)
	}
	defer rows.Close()

	return scanRows(schema, rows)
}

func (d *Database) Count(ctx context.Context, schema *model.Schema, opts ...model.QueryOption) (int64, error) {
	q := model.ApplyQueryOptions(opts...)

	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", quoteIdent(schema.Table))
	var args []any
	paramN := 1

	if len(q.Filters) > 0 {
		where, fArgs, _ := buildWhere(q.Filters, paramN)
		query += " WHERE " + where
		args = append(args, fArgs...)
	}

	var count int64
	err := d.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("model/postgres: count: %w", err)
	}
	return count, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) String() string {
	return "postgres"
}

// SQL helpers

func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func goTypeToPostgres(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Int, reflect.Int64:
		return "BIGINT"
	case reflect.Int8, reflect.Int16, reflect.Int32:
		return "INTEGER"
	case reflect.Uint, reflect.Uint64:
		return "BIGINT"
	case reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return "INTEGER"
	case reflect.Float32:
		return "REAL"
	case reflect.Float64:
		return "DOUBLE PRECISION"
	case reflect.Bool:
		return "BOOLEAN"
	default:
		return "TEXT"
	}
}

func buildInsert(schema *model.Schema, fields map[string]any) (string, string, []any) {
	var cols []string
	var placeholders []string
	var values []any
	i := 1
	for _, f := range schema.Fields {
		if v, ok := fields[f.Column]; ok {
			cols = append(cols, quoteIdent(f.Column))
			placeholders = append(placeholders, fmt.Sprintf("$%d", i))
			values = append(values, v)
			i++
		}
	}
	return strings.Join(cols, ", "), strings.Join(placeholders, ", "), values
}

func buildUpdate(schema *model.Schema, fields map[string]any) (string, []any) {
	var setClauses []string
	var values []any
	i := 1
	for _, f := range schema.Fields {
		if f.IsKey {
			continue
		}
		if v, ok := fields[f.Column]; ok {
			setClauses = append(setClauses, fmt.Sprintf("%s = $%d", quoteIdent(f.Column), i))
			values = append(values, v)
			i++
		}
	}
	return strings.Join(setClauses, ", "), values
}

func buildWhere(filters []model.Filter, startParam int) (string, []any, int) {
	var clauses []string
	var args []any
	n := startParam
	for _, f := range filters {
		clauses = append(clauses, fmt.Sprintf("%s %s $%d", quoteIdent(f.Field), f.Op, n))
		args = append(args, f.Value)
		n++
	}
	return strings.Join(clauses, " AND "), args, n
}

func columnList(schema *model.Schema) string {
	var cols []string
	for _, f := range schema.Fields {
		cols = append(cols, quoteIdent(f.Column))
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
		return nil, fmt.Errorf("model/postgres: scan: %w", err)
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
			return nil, fmt.Errorf("model/postgres: scan row: %w", err)
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
