package orm

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Migration represents a database migration step.
type Migration struct {
	Version string // Timestamp or unique version string e.g. "20260914120000"
	Name    string // Descriptive name e.g. "create_users_table"
	Up      string // SQL statements to apply
	Down    string // SQL statements to revert
}

// MigrationStatus describes the applied state of a migration.
type MigrationStatus struct {
	Version   string
	Name      string
	Applied   bool
	Batch     int
	AppliedAt *time.Time
}

// Migrate applies all pending migrations in order inside individual transactions.
// It creates a 'goks_migrations' tracking table if it doesn't exist.
func Migrate(db *Database, migrations []Migration) error {
	if db == nil {
		db = DB
	}
	if db == nil {
		return fmt.Errorf("goks/orm: no active database connection")
	}

	if err := ensureMigrationsTable(db); err != nil {
		return fmt.Errorf("goks/orm: ensure migrations table: %w", err)
	}

	// 1. Get current max batch
	var maxBatch int
	row := db.Raw().QueryRow("SELECT COALESCE(MAX(batch), 0) FROM goks_migrations")
	if err := row.Scan(&maxBatch); err != nil {
		maxBatch = 0
	}
	nextBatch := maxBatch + 1

	// 2. Query applied migration versions
	rows, err := db.Raw().Query("SELECT version FROM goks_migrations")
	if err != nil {
		return fmt.Errorf("goks/orm: query applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err == nil {
			applied[v] = true
		}
	}

	// 3. Run pending migrations in individual transactions
	p1 := placeholder(db, 1)
	p2 := placeholder(db, 2)
	p3 := placeholder(db, 3)
	insertQuery := fmt.Sprintf("INSERT INTO goks_migrations (version, name, batch) VALUES (%s, %s, %s)", p1, p2, p3)

	for _, m := range migrations {
		key := m.Version
		if key == "" {
			key = m.Name
		}
		if applied[key] || applied[m.Name] {
			continue
		}

		tx, err := db.Raw().Begin()
		if err != nil {
			return fmt.Errorf("goks/orm: begin transaction for %s: %w", m.Name, err)
		}

		if strings.TrimSpace(m.Up) != "" {
			if _, err := tx.Exec(m.Up); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("goks/orm: migration %s failed: %w", m.Name, err)
			}
		}

		if _, err := tx.Exec(insertQuery, key, m.Name, nextBatch); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("goks/orm: record migration %s: %w", m.Name, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("goks/orm: commit migration %s: %w", m.Name, err)
		}
	}

	return nil
}

// Rollback reverts the last batch of migrations (or last N batches if steps > 1).
func Rollback(db *Database, migrations []Migration, steps int) error {
	if db == nil {
		db = DB
	}
	if db == nil {
		return fmt.Errorf("goks/orm: no active database connection")
	}

	if err := ensureMigrationsTable(db); err != nil {
		return err
	}

	if steps <= 0 {
		steps = 1
	}

	migMap := make(map[string]Migration)
	for _, m := range migrations {
		key := m.Version
		if key == "" {
			key = m.Name
		}
		migMap[key] = m
		migMap[m.Name] = m
	}

	p1 := placeholder(db, 1)
	delQuery := fmt.Sprintf("DELETE FROM goks_migrations WHERE version = %s", p1)

	for s := 0; s < steps; s++ {
		var maxBatch int
		row := db.Raw().QueryRow("SELECT COALESCE(MAX(batch), 0) FROM goks_migrations")
		if err := row.Scan(&maxBatch); err != nil || maxBatch <= 0 {
			break // No more batches to revert
		}

		batchPlaceholder := placeholder(db, 1)
		batchQuery := fmt.Sprintf("SELECT version, name FROM goks_migrations WHERE batch = %s ORDER BY id DESC", batchPlaceholder)
		rows, err := db.Raw().Query(batchQuery, maxBatch)
		if err != nil {
			return fmt.Errorf("goks/orm: query batch %d: %w", maxBatch, err)
		}

		type appliedItem struct {
			version string
			name    string
		}
		var toRevert []appliedItem
		for rows.Next() {
			var item appliedItem
			if err := rows.Scan(&item.version, &item.name); err == nil {
				toRevert = append(toRevert, item)
			}
		}
		rows.Close()

		for _, item := range toRevert {
			m, ok := migMap[item.version]
			if !ok {
				m = migMap[item.name]
			}

			tx, err := db.Raw().Begin()
			if err != nil {
				return err
			}

			if strings.TrimSpace(m.Down) != "" {
				if _, err := tx.Exec(m.Down); err != nil {
					_ = tx.Rollback()
					return fmt.Errorf("goks/orm: rollback %s failed: %w", item.name, err)
				}
			}

			if _, err := tx.Exec(delQuery, item.version); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("goks/orm: clear migration record %s: %w", item.name, err)
			}

			if err := tx.Commit(); err != nil {
				return err
			}
		}
	}

	return nil
}

// GetStatus returns the current execution status of all migrations.
func GetStatus(db *Database, migrations []Migration) ([]MigrationStatus, error) {
	if db == nil {
		db = DB
	}
	if db == nil {
		return nil, fmt.Errorf("goks/orm: no active database connection")
	}

	if err := ensureMigrationsTable(db); err != nil {
		return nil, err
	}

	rows, err := db.Raw().Query("SELECT version, batch, applied_at FROM goks_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type record struct {
		batch     int
		appliedAt time.Time
	}
	appliedMap := make(map[string]record)
	for rows.Next() {
		var v string
		var b int
		var t time.Time
		if err := rows.Scan(&v, &b, &t); err == nil {
			appliedMap[v] = record{batch: b, appliedAt: t}
		}
	}

	var statuses []MigrationStatus
	for _, m := range migrations {
		key := m.Version
		if key == "" {
			key = m.Name
		}
		rec, isApplied := appliedMap[key]
		if !isApplied {
			rec, isApplied = appliedMap[m.Name]
		}

		st := MigrationStatus{
			Version: m.Version,
			Name:    m.Name,
			Applied: isApplied,
		}
		if isApplied {
			st.Batch = rec.batch
			t := rec.appliedAt
			st.AppliedAt = &t
		}
		statuses = append(statuses, st)
	}

	return statuses, nil
}

// ParseMigrationFile reads a .sql migration file and extracts Up and Down sections.
func ParseMigrationFile(filePath string) (*Migration, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	base := filepath.Base(filePath)
	nameWithoutExt := strings.TrimSuffix(base, filepath.Ext(base))
	parts := strings.SplitN(nameWithoutExt, "_", 2)
	version := parts[0]
	name := nameWithoutExt
	if len(parts) > 1 {
		name = parts[1]
	}

	content := string(data)
	var upLines, downLines []string
	isUp := false
	isDown := false

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.EqualFold(trimmed, "-- +goks Up") || strings.EqualFold(trimmed, "-- +goose Up") {
			isUp = true
			isDown = false
			continue
		}
		if strings.EqualFold(trimmed, "-- +goks Down") || strings.EqualFold(trimmed, "-- +goose Down") {
			isUp = false
			isDown = true
			continue
		}

		if isDown {
			downLines = append(downLines, line)
		} else if isUp {
			upLines = append(upLines, line)
		} else {
			// If no explicit markers yet, default to Up
			upLines = append(upLines, line)
		}
	}

	return &Migration{
		Version: version,
		Name:    name,
		Up:      strings.TrimSpace(strings.Join(upLines, "\n")),
		Down:    strings.TrimSpace(strings.Join(downLines, "\n")),
	}, nil
}

// LoadMigrationsFromDir loads all .sql migration files from the directory, sorted by version.
func LoadMigrationsFromDir(dir string) ([]Migration, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var migrations []Migration
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		m, err := ParseMigrationFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to parse migration %s: %w", entry.Name(), err)
		}
		migrations = append(migrations, *m)
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// RunSeeders executes SQL seed files located in seeds/ directory in transactions.
func RunSeeders(db *Database, seedsDir string) ([]string, error) {
	if db == nil {
		db = DB
	}
	if db == nil {
		return nil, fmt.Errorf("goks/orm: no active database connection")
	}

	if _, err := os.Stat(seedsDir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(seedsDir)
	if err != nil {
		return nil, err
	}

	var executed []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(seedsDir, entry.Name()))
		if err != nil {
			return executed, err
		}

		tx, err := db.Raw().Begin()
		if err != nil {
			return executed, err
		}

		if _, err := tx.Exec(string(data)); err != nil {
			_ = tx.Rollback()
			return executed, fmt.Errorf("seeder %s failed: %w", entry.Name(), err)
		}

		if err := tx.Commit(); err != nil {
			return executed, err
		}
		executed = append(executed, entry.Name())
	}

	return executed, nil
}

func ensureMigrationsTable(db *Database) error {
	dialectName := "sqlite"
	if db != nil && db.dialect != nil {
		dialectName = db.dialect.Name()
	}

	var schema string
	switch dialectName {
	case "postgres":
		schema = `CREATE TABLE IF NOT EXISTS goks_migrations (
			id SERIAL PRIMARY KEY,
			version VARCHAR(255) NOT NULL UNIQUE,
			name VARCHAR(255) NOT NULL,
			batch INTEGER NOT NULL,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`
	case "mysql":
		schema = `CREATE TABLE IF NOT EXISTS goks_migrations (
			id INT AUTO_INCREMENT PRIMARY KEY,
			version VARCHAR(255) NOT NULL UNIQUE,
			name VARCHAR(255) NOT NULL,
			batch INT NOT NULL,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`
	default: // sqlite
		schema = `CREATE TABLE IF NOT EXISTS goks_migrations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			version TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			batch INTEGER NOT NULL,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`
	}

	_, err := db.Raw().Exec(schema)
	return err
}

func placeholder(db *Database, n int) string {
	if db != nil && db.dialect != nil {
		return db.dialect.Placeholder(n)
	}
	return "?"
}

// CreateTable generates a simple CREATE TABLE SQL from a model struct.
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
