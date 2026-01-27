// Package db provides a database abstraction layer for go-micro services.
// It follows the repository pattern like Spring Data and ActiveRecord.
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

var (
	// ErrNotFound is returned when a record is not found.
	ErrNotFound = errors.New("record not found")
	// ErrDuplicate is returned when a duplicate record exists.
	ErrDuplicate = errors.New("duplicate record")
)

// DB is the database interface.
type DB interface {
	// Exec executes a query without returning rows.
	Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	// Query executes a query that returns rows.
	Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	// QueryRow executes a query that returns a single row.
	QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row
	// Begin starts a transaction.
	Begin(ctx context.Context) (Tx, error)
	// Close closes the database connection.
	Close() error
	// Ping verifies the connection is alive.
	Ping(ctx context.Context) error
}

// Tx is a database transaction.
type Tx interface {
	Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row
	Commit() error
	Rollback() error
}

// Config holds database configuration.
type Config struct {
	Driver          string        // postgres, mysql, sqlite3
	DSN             string        // Data source name
	MaxOpenConns    int           // Max open connections
	MaxIdleConns    int           // Max idle connections
	ConnMaxLifetime time.Duration // Connection max lifetime
}

// Option configures the database.
type Option func(*Config)

// WithDriver sets the database driver.
func WithDriver(driver string) Option {
	return func(c *Config) { c.Driver = driver }
}

// WithDSN sets the data source name.
func WithDSN(dsn string) Option {
	return func(c *Config) { c.DSN = dsn }
}

// WithMaxOpenConns sets max open connections.
func WithMaxOpenConns(n int) Option {
	return func(c *Config) { c.MaxOpenConns = n }
}

// WithMaxIdleConns sets max idle connections.
func WithMaxIdleConns(n int) Option {
	return func(c *Config) { c.MaxIdleConns = n }
}

// WithConnMaxLifetime sets connection max lifetime.
func WithConnMaxLifetime(d time.Duration) Option {
	return func(c *Config) { c.ConnMaxLifetime = d }
}

type database struct {
	db     *sql.DB
	config Config
}

type transaction struct {
	tx *sql.Tx
}

var (
	defaultDB   DB
	defaultOnce sync.Once
)

// Default returns the default database connection.
func Default() DB {
	return defaultDB
}

// SetDefault sets the default database connection.
func SetDefault(db DB) {
	defaultOnce.Do(func() {
		defaultDB = db
	})
}

// New creates a new database connection.
func New(opts ...Option) (DB, error) {
	config := Config{
		Driver:          "postgres",
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
	}

	// Apply options
	for _, opt := range opts {
		opt(&config)
	}

	// Use environment variables as fallback
	if config.DSN == "" {
		config.DSN = os.Getenv("DATABASE_URL")
	}
	if config.Driver == "" {
		if driver := os.Getenv("DATABASE_DRIVER"); driver != "" {
			config.Driver = driver
		}
	}

	if config.DSN == "" {
		return nil, fmt.Errorf("database DSN required: set DATABASE_URL or use WithDSN()")
	}

	db, err := sql.Open(config.Driver, config.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &database{db: db, config: config}, nil
}

func (d *database) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return d.db.ExecContext(ctx, query, args...)
}

func (d *database) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return d.db.QueryContext(ctx, query, args...)
}

func (d *database) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return d.db.QueryRowContext(ctx, query, args...)
}

func (d *database) Begin(ctx context.Context) (Tx, error) {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &transaction{tx: tx}, nil
}

func (d *database) Close() error {
	return d.db.Close()
}

func (d *database) Ping(ctx context.Context) error {
	return d.db.PingContext(ctx)
}

func (t *transaction) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return t.tx.ExecContext(ctx, query, args...)
}

func (t *transaction) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return t.tx.QueryContext(ctx, query, args...)
}

func (t *transaction) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return t.tx.QueryRowContext(ctx, query, args...)
}

func (t *transaction) Commit() error {
	return t.tx.Commit()
}

func (t *transaction) Rollback() error {
	return t.tx.Rollback()
}
