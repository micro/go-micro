// Package migrate provides database migration support for go-micro.
// Similar to Rails migrations or Flyway.
package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Migration represents a database migration.
type Migration struct {
	Version     int64
	Name        string
	Up          func(ctx context.Context, tx *sql.Tx) error
	Down        func(ctx context.Context, tx *sql.Tx) error
	UpSQL       string // Alternative: raw SQL
	DownSQL     string
}

// Migrator handles database migrations.
type Migrator struct {
	db         *sql.DB
	table      string
	migrations []Migration
}

// Option configures the migrator.
type Option func(*Migrator)

// WithTable sets the migrations table name.
func WithTable(table string) Option {
	return func(m *Migrator) { m.table = table }
}

// New creates a new migrator.
func New(db *sql.DB, opts ...Option) *Migrator {
	m := &Migrator{
		db:    db,
		table: "schema_migrations",
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// Register adds a migration.
func (m *Migrator) Register(migration Migration) {
	m.migrations = append(m.migrations, migration)
}

// Init creates the migrations table if it doesn't exist.
func (m *Migrator) Init(ctx context.Context) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			version BIGINT PRIMARY KEY,
			name VARCHAR(255),
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`, m.table)
	_, err := m.db.ExecContext(ctx, query)
	return err
}

// Up runs all pending migrations.
func (m *Migrator) Up(ctx context.Context) error {
	if err := m.Init(ctx); err != nil {
		return fmt.Errorf("failed to init migrations table: %w", err)
	}

	applied, err := m.appliedVersions(ctx)
	if err != nil {
		return err
	}

	// Sort migrations by version
	sort.Slice(m.migrations, func(i, j int) bool {
		return m.migrations[i].Version < m.migrations[j].Version
	})

	for _, migration := range m.migrations {
		if applied[migration.Version] {
			continue
		}

		fmt.Printf("Running migration %d: %s\n", migration.Version, migration.Name)

		if err := m.runMigration(ctx, migration, true); err != nil {
			return fmt.Errorf("migration %d failed: %w", migration.Version, err)
		}
	}

	return nil
}

// Down rolls back the last migration.
func (m *Migrator) Down(ctx context.Context) error {
	if err := m.Init(ctx); err != nil {
		return fmt.Errorf("failed to init migrations table: %w", err)
	}

	// Get the last applied migration
	var version int64
	var name string
	query := fmt.Sprintf("SELECT version, name FROM %s ORDER BY version DESC LIMIT 1", m.table)
	err := m.db.QueryRowContext(ctx, query).Scan(&version, &name)
	if err == sql.ErrNoRows {
		fmt.Println("No migrations to rollback")
		return nil
	}
	if err != nil {
		return err
	}

	// Find the migration
	var migration *Migration
	for i := range m.migrations {
		if m.migrations[i].Version == version {
			migration = &m.migrations[i]
			break
		}
	}

	if migration == nil {
		return fmt.Errorf("migration %d not found in registered migrations", version)
	}

	fmt.Printf("Rolling back migration %d: %s\n", version, name)
	return m.runMigration(ctx, *migration, false)
}

// Status prints the migration status.
func (m *Migrator) Status(ctx context.Context) error {
	if err := m.Init(ctx); err != nil {
		return err
	}

	applied, err := m.appliedVersions(ctx)
	if err != nil {
		return err
	}

	fmt.Printf("%-15s %-40s %s\n", "VERSION", "NAME", "STATUS")
	fmt.Println(strings.Repeat("-", 70))

	sort.Slice(m.migrations, func(i, j int) bool {
		return m.migrations[i].Version < m.migrations[j].Version
	})

	for _, migration := range m.migrations {
		status := "pending"
		if applied[migration.Version] {
			status = "applied"
		}
		fmt.Printf("%-15d %-40s %s\n", migration.Version, migration.Name, status)
	}

	return nil
}

func (m *Migrator) runMigration(ctx context.Context, migration Migration, up bool) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if up {
		if migration.Up != nil {
			if err := migration.Up(ctx, tx); err != nil {
				return err
			}
		} else if migration.UpSQL != "" {
			if _, err := tx.ExecContext(ctx, migration.UpSQL); err != nil {
				return err
			}
		}

		// Record migration
		query := fmt.Sprintf("INSERT INTO %s (version, name) VALUES ($1, $2)", m.table)
		if _, err := tx.ExecContext(ctx, query, migration.Version, migration.Name); err != nil {
			return err
		}
	} else {
		if migration.Down != nil {
			if err := migration.Down(ctx, tx); err != nil {
				return err
			}
		} else if migration.DownSQL != "" {
			if _, err := tx.ExecContext(ctx, migration.DownSQL); err != nil {
				return err
			}
		}

		// Remove migration record
		query := fmt.Sprintf("DELETE FROM %s WHERE version = $1", m.table)
		if _, err := tx.ExecContext(ctx, query, migration.Version); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (m *Migrator) appliedVersions(ctx context.Context) (map[int64]bool, error) {
	applied := make(map[int64]bool)

	query := fmt.Sprintf("SELECT version FROM %s", m.table)
	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var version int64
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}

	return applied, rows.Err()
}

// GenerateVersion generates a migration version based on timestamp.
func GenerateVersion() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// LoadFromDir loads SQL migrations from a directory.
// Files should be named: {version}_{name}.up.sql and {version}_{name}.down.sql
func LoadFromDir(dir string) ([]Migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	migrations := make(map[int64]*Migration)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}

		// Parse filename: {version}_{name}.{up|down}.sql
		parts := strings.SplitN(name, "_", 2)
		if len(parts) < 2 {
			continue
		}

		version, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			continue
		}

		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}

		if migrations[version] == nil {
			migrations[version] = &Migration{Version: version}
		}

		if strings.Contains(name, ".up.") {
			migrations[version].UpSQL = string(content)
			// Extract name from filename
			migrations[version].Name = strings.TrimSuffix(parts[1], ".up.sql")
		} else if strings.Contains(name, ".down.") {
			migrations[version].DownSQL = string(content)
		}
	}

	result := make([]Migration, 0, len(migrations))
	for _, m := range migrations {
		result = append(result, *m)
	}

	return result, nil
}
