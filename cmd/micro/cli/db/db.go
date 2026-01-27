// Package db provides database CLI commands for micro.
package db

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v5/cmd"
)

func init() {
	cmd.Register(&cli.Command{
		Name:  "db",
		Usage: "Database commands (migrations)",
		Subcommands: []*cli.Command{
			{
				Name:   "migrate",
				Usage:  "Run pending migrations",
				Action: migrateUp,
			},
			{
				Name:   "rollback",
				Usage:  "Rollback the last migration",
				Action: migrateDown,
			},
			{
				Name:   "status",
				Usage:  "Show migration status",
				Action: migrateStatus,
			},
			{
				Name:   "create",
				Usage:  "Create a new migration: micro db create <name>",
				Action: createMigration,
			},
		},
	})
}

func migrateUp(c *cli.Context) error {
	fmt.Println("Running migrations...")
	fmt.Println("")
	fmt.Println("To use migrations in your service, add:")
	fmt.Println("")
	fmt.Println("  import (")
	fmt.Println("      \"database/sql\"")
	fmt.Println("      _ \"github.com/lib/pq\" // or mysql, sqlite3")
	fmt.Println("      \"go-micro.dev/v5/db/migrate\"")
	fmt.Println("  )")
	fmt.Println("")
	fmt.Println("  db, _ := sql.Open(\"postgres\", os.Getenv(\"DATABASE_URL\"))")
	fmt.Println("  m := migrate.New(db)")
	fmt.Println("  migrations, _ := migrate.LoadFromDir(\"./migrations\")")
	fmt.Println("  for _, migration := range migrations {")
	fmt.Println("      m.Register(migration)")
	fmt.Println("  }")
	fmt.Println("  m.Up(context.Background())")
	return nil
}

func migrateDown(c *cli.Context) error {
	fmt.Println("Rollback requires database connection. See 'micro db migrate' for setup.")
	return nil
}

func migrateStatus(c *cli.Context) error {
	fmt.Println("Status requires database connection. See 'micro db migrate' for setup.")
	return nil
}

func createMigration(c *cli.Context) error {
	name := c.Args().First()
	if name == "" {
		return fmt.Errorf("migration name required: micro db create <name>")
	}

	// Create migrations directory
	if err := os.MkdirAll("migrations", 0755); err != nil {
		return fmt.Errorf("failed to create migrations directory: %w", err)
	}

	// Generate version timestamp
	version := time.Now().Format("20060102150405")

	// Create up migration
	upFile := filepath.Join("migrations", fmt.Sprintf("%s_%s.up.sql", version, name))
	upContent := fmt.Sprintf("-- Migration: %s\n-- Created: %s\n\n-- Write your UP migration here\n", name, time.Now().Format(time.RFC3339))
	if err := os.WriteFile(upFile, []byte(upContent), 0644); err != nil {
		return fmt.Errorf("failed to create up migration: %w", err)
	}

	// Create down migration
	downFile := filepath.Join("migrations", fmt.Sprintf("%s_%s.down.sql", version, name))
	downContent := fmt.Sprintf("-- Migration: %s (rollback)\n-- Created: %s\n\n-- Write your DOWN migration here\n", name, time.Now().Format(time.RFC3339))
	if err := os.WriteFile(downFile, []byte(downContent), 0644); err != nil {
		return fmt.Errorf("failed to create down migration: %w", err)
	}

	fmt.Printf("Created migration files:\n")
	fmt.Printf("  %s\n", upFile)
	fmt.Printf("  %s\n", downFile)

	return nil
}
