// Package orm provides the GoKS ORM layer.
// It wraps database/sql with a fluent query builder and model conventions.
package orm

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// Model is the base struct to embed in all GoKS ORM models.
// It provides ID, CreatedAt, UpdatedAt, and DeletedAt fields.
//
//	type User struct {
//	    orm.Model
//	    Name  string
//	    Email string
//	}
type Model struct {
	ID        uint      `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at"` // soft-delete support
}

// IsNew returns true if the model has not been saved to the database yet.
func (m *Model) IsNew() bool {
	return m.ID == 0
}

// -----------------------------------------------------------------------
// DB — the global database connection
// -----------------------------------------------------------------------

// DB is the global GoKS database handle.
var DB *Database

// Database wraps *sql.DB with GoKS query builder methods.
type Database struct {
	db      *sql.DB
	dialect Dialect
}

// Connect opens a database connection.
// driver: "postgres", "mysql", "sqlite3"
func Connect(driver, dsn string) (*Database, error) {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("goks/orm: connect: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("goks/orm: ping: %w", err)
	}

	d := &Database{db: db, dialect: dialectFor(driver)}
	DB = d
	return d, nil
}

// Raw returns the underlying *sql.DB.
func (d *Database) Raw() *sql.DB { return d.db }

// Close closes the database connection.
func (d *Database) Close() error { return d.db.Close() }

// -----------------------------------------------------------------------
// Query builder
// -----------------------------------------------------------------------

// Query starts a new query builder for the given model type.
func Query[T any](db ...*Database) *Builder[T] {
	target := new(T)
	tableName := tableNameOf(target)
	d := DB
	if len(db) > 0 && db[0] != nil {
		d = db[0]
	}
	return &Builder[T]{db: d, table: tableName}
}

// Builder is a fluent query builder for a specific model type.
type Builder[T any] struct {
	db         *Database
	table      string
	conditions []string
	args       []any
	orderBy    string
	limitVal   int
	offsetVal  int
}

// Where adds a WHERE condition. Conditions are ANDed together.
func (b *Builder[T]) Where(condition string, args ...any) *Builder[T] {
	b.conditions = append(b.conditions, condition)
	b.args = append(b.args, args...)
	return b
}

// OrderBy sets the ORDER BY clause.
func (b *Builder[T]) OrderBy(col string) *Builder[T] {
	b.orderBy = col
	return b
}

// Limit sets the LIMIT.
func (b *Builder[T]) Limit(n int) *Builder[T] {
	b.limitVal = n
	return b
}

// Offset sets the OFFSET.
func (b *Builder[T]) Offset(n int) *Builder[T] {
	b.offsetVal = n
	return b
}

// Find executes a SELECT and returns all matching rows.
func (b *Builder[T]) Find() ([]T, error) {
	query := b.buildSelect("*")
	rows, err := b.db.db.Query(query, b.args...)
	if err != nil {
		return nil, fmt.Errorf("goks/orm: find: %w", err)
	}
	defer rows.Close()
	return scanRows[T](rows)
}

// First returns the first matching row.
func (b *Builder[T]) First() (*T, error) {
	b.limitVal = 1
	rows, err := b.Limit(1).Find()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, sql.ErrNoRows
	}
	return &rows[0], nil
}

// Count returns the number of matching rows.
func (b *Builder[T]) Count() (int64, error) {
	query := b.buildSelect("COUNT(*)")
	var count int64
	err := b.db.db.QueryRow(query, b.args...).Scan(&count)
	return count, err
}

// buildSelect constructs the SELECT SQL string.
func (b *Builder[T]) buildSelect(cols string) string {
	q := fmt.Sprintf("SELECT %s FROM %s", cols, b.table)
	// Filter soft-deleted rows by default
	conditions := append([]string{"deleted_at IS NULL"}, b.conditions...)
	if len(conditions) > 0 {
		q += " WHERE " + strings.Join(conditions, " AND ")
	}
	if b.orderBy != "" {
		q += " ORDER BY " + b.orderBy
	}
	if b.limitVal > 0 {
		q += fmt.Sprintf(" LIMIT %d", b.limitVal)
	}
	if b.offsetVal > 0 {
		q += fmt.Sprintf(" OFFSET %d", b.offsetVal)
	}
	return q
}


// -----------------------------------------------------------------------
// Introspection helpers
// -----------------------------------------------------------------------

// tableNameOf derives the table name from a model struct type.
// e.g. *User → "users"
func tableNameOf(v any) string {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	name := t.Name()
	// Simple pluralisation: append 's' (good enough for MVP)
	return strings.ToLower(name) + "s"
}

// scanRows scans sql.Rows into a slice of T using reflection.
func scanRows[T any](rows *sql.Rows) ([]T, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var results []T
	for rows.Next() {
		var item T
		ptrs := fieldPointers(&item, cols)
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		results = append(results, item)
	}
	return results, rows.Err()
}

// fieldPointers returns scan destination pointers for struct fields matching cols.
func fieldPointers(v any, cols []string) []any {
	rv := reflect.ValueOf(v).Elem()
	rt := rv.Type()
	tagIndex := make(map[string]int)
	for i := 0; i < rt.NumField(); i++ {
		tag := rt.Field(i).Tag.Get("db")
		if tag == "" {
			tag = strings.ToLower(rt.Field(i).Name)
		}
		tagIndex[tag] = i
	}
	ptrs := make([]any, len(cols))
	for i, col := range cols {
		if idx, ok := tagIndex[col]; ok {
			ptrs[i] = rv.Field(idx).Addr().Interface()
		} else {
			// discard unknown column
			var discard any
			ptrs[i] = &discard
		}
	}
	return ptrs
}

// -----------------------------------------------------------------------
// Dialect
// -----------------------------------------------------------------------

// Dialect abstracts SQL differences between databases.
type Dialect interface {
	Placeholder(n int) string // e.g. $1 (postgres) or ? (mysql/sqlite)
}

type postgresDialect struct{}
type mysqlDialect struct{}

func (postgresDialect) Placeholder(n int) string { return fmt.Sprintf("$%d", n) }
func (mysqlDialect) Placeholder(_ int) string    { return "?" }

func dialectFor(driver string) Dialect {
	switch driver {
	case "postgres", "pgx":
		return postgresDialect{}
	default:
		return mysqlDialect{}
	}
}
