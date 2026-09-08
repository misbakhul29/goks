package orm

import (
	"database/sql"
	"fmt"
	"strings"
)

// MigrateConfig holds the configuration for a migration.
type MigrateConfig struct {
	DB *Database
}

// Migration represents a database migration step.
type Migration struct {
	Name string
	Up   string // SQL to apply
	Down string // SQL to revert
}

// Migrate applies all pending migrations in order.
// It creates a 'goks_migrations' tracking table if it doesn't exist.
func Migrate(db *Database, migrations []Migration) error {
	if db == nil {
		db = DB
	}
	if err := ensureMigrationsTable(db.db); err != nil {
		return err
	}

	applied, err := appliedMigrations(db.db)
	if err != nil {
		return err
	}

	for _, m := range migrations {
		if applied[m.Name] {
			continue
		}
		if _, err := db.db.Exec(m.Up); err != nil {
			return fmt.Errorf("goks/orm: migration %q failed: %w", m.Name, err)
		}
		if _, err := db.db.Exec(
			"INSERT INTO goks_migrations (name) VALUES ($1)", m.Name,
		); err != nil {
			return fmt.Errorf("goks/orm: recording migration %q: %w", m.Name, err)
		}
	}
	return nil
}

// Rollback reverts the last N migrations.
func Rollback(db *Database, migrations []Migration, steps int) error {
	if db == nil {
		db = DB
	}
	applied, err := appliedMigrations(db.db)
	if err != nil {
		return err
	}

	reverted := 0
	for i := len(migrations) - 1; i >= 0 && reverted < steps; i-- {
		m := migrations[i]
		if !applied[m.Name] {
			continue
		}
		if _, err := db.db.Exec(m.Down); err != nil {
			return fmt.Errorf("goks/orm: rollback %q failed: %w", m.Name, err)
		}
		if _, err := db.db.Exec(
			"DELETE FROM goks_migrations WHERE name = $1", m.Name,
		); err != nil {
			return fmt.Errorf("goks/orm: clearing migration record %q: %w", m.Name, err)
		}
		reverted++
	}
	return nil
}

// CreateTable generates a simple CREATE TABLE SQL from a model struct.
// Use this as a helper to build Migration.Up strings quickly.
func CreateTable[T any]() string {
	var zero T
	table := tableNameOf(&zero)
	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n  id SERIAL PRIMARY KEY,\n  created_at TIMESTAMP NOT NULL DEFAULT NOW(),\n  updated_at TIMESTAMP NOT NULL DEFAULT NOW(),\n  deleted_at TIMESTAMP NULL\n);", table)
}

// DropTable generates a DROP TABLE SQL for a model.
func DropTable[T any]() string {
	var zero T
	table := tableNameOf(&zero)
	return fmt.Sprintf("DROP TABLE IF EXISTS %s;", table)
}

// AddColumn generates an ALTER TABLE ... ADD COLUMN statement.
func AddColumn(table, column, colType string) string {
	return fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS %s %s;", table, column, colType)
}

// CreateIndex generates a CREATE INDEX statement.
func CreateIndex(table string, cols ...string) string {
	idxName := fmt.Sprintf("idx_%s_%s", table, strings.Join(cols, "_"))
	return fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (%s);", idxName, table, strings.Join(cols, ", "))
}

func ensureMigrationsTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS goks_migrations (
			id         SERIAL PRIMARY KEY,
			name       TEXT NOT NULL UNIQUE,
			applied_at TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`)
	return err
}

func appliedMigrations(db *sql.DB) (map[string]bool, error) {
	rows, err := db.Query("SELECT name FROM goks_migrations")
	if err != nil {
		return nil, fmt.Errorf("goks/orm: reading migrations: %w", err)
	}
	defer rows.Close()
	result := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		result[name] = true
	}
	return result, rows.Err()
}
