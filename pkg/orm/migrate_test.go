package orm_test

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/misbakhul29/goks/pkg/orm"
)

var (
	testMockDrv  = &mockDriver{}
	registerOnce sync.Once
)

func init() {
	registerOnce.Do(func() {
		sql.Register("mock_sqlite", testMockDrv)
	})
}

type mockDriver struct {
	mu      sync.Mutex
	queries []string
	failing bool
}

func (d *mockDriver) Open(name string) (driver.Conn, error) {
	return &mockConn{driver: d}, nil
}

func (d *mockDriver) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.queries = nil
	d.failing = false
}

type mockConn struct {
	driver *mockDriver
}

func (c *mockConn) Prepare(query string) (driver.Stmt, error) {
	c.driver.mu.Lock()
	c.driver.queries = append(c.driver.queries, query)
	c.driver.mu.Unlock()
	return &mockStmt{conn: c, query: query}, nil
}

func (c *mockConn) Close() error { return nil }

func (c *mockConn) Begin() (driver.Tx, error) {
	return &mockTx{conn: c}, nil
}

type mockTx struct {
	conn *mockConn
}

func (tx *mockTx) Commit() error   { return nil }
func (tx *mockTx) Rollback() error { return nil }

type mockStmt struct {
	conn  *mockConn
	query string
}

func (s *mockStmt) Close() error  { return nil }
func (s *mockStmt) NumInput() int { return -1 }

func (s *mockStmt) Exec(args []driver.Value) (driver.Result, error) {
	s.conn.driver.mu.Lock()
	defer s.conn.driver.mu.Unlock()
	if s.conn.driver.failing && strings.Contains(s.query, "INVALID") {
		return nil, fmt.Errorf("mock execution failure")
	}
	s.conn.driver.queries = append(s.conn.driver.queries, s.query)
	return driver.RowsAffected(1), nil
}

func (s *mockStmt) Query(args []driver.Value) (driver.Rows, error) {
	s.conn.driver.mu.Lock()
	defer s.conn.driver.mu.Unlock()
	s.conn.driver.queries = append(s.conn.driver.queries, s.query)

	if strings.Contains(s.query, "MAX(batch)") {
		return &mockRows{cols: []string{"batch"}, vals: [][]driver.Value{{int64(0)}}}, nil
	}
	if strings.Contains(s.query, "SELECT version FROM goks_migrations") {
		return &mockRows{cols: []string{"version"}, vals: [][]driver.Value{}}, nil
	}
	if strings.Contains(s.query, "SELECT version, batch, applied_at FROM goks_migrations") {
		return &mockRows{cols: []string{"version", "batch", "applied_at"}, vals: [][]driver.Value{}}, nil
	}
	return &mockRows{cols: []string{"id"}, vals: nil}, nil
}

type mockRows struct {
	cols []string
	vals [][]driver.Value
	idx  int
}

func (r *mockRows) Columns() []string { return r.cols }
func (r *mockRows) Close() error      { return nil }
func (r *mockRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.vals) {
		return io.EOF
	}
	row := r.vals[r.idx]
	copy(dest, row)
	r.idx++
	return nil
}

func TestParseMigrationFile(t *testing.T) {
	tempDir := t.TempDir()
	migFile := filepath.Join(tempDir, "20260914120000_create_posts_table.sql")

	content := `-- +goks Up
CREATE TABLE posts (
    id INTEGER PRIMARY KEY,
    title TEXT NOT NULL
);

-- +goks Down
DROP TABLE IF EXISTS posts;
`
	if err := os.WriteFile(migFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write migration file: %v", err)
	}

	mig, err := orm.ParseMigrationFile(migFile)
	if err != nil {
		t.Fatalf("ParseMigrationFile failed: %v", err)
	}

	if mig.Version != "20260914120000" {
		t.Errorf("expected version 20260914120000, got %s", mig.Version)
	}
	if mig.Name != "create_posts_table" {
		t.Errorf("expected name create_posts_table, got %s", mig.Name)
	}
	if !strings.Contains(mig.Up, "CREATE TABLE posts") {
		t.Errorf("expected Up to contain CREATE TABLE posts, got:\n%s", mig.Up)
	}
	if !strings.Contains(mig.Down, "DROP TABLE IF EXISTS posts") {
		t.Errorf("expected Down to contain DROP TABLE, got:\n%s", mig.Down)
	}
}

func TestLoadMigrationsFromDir(t *testing.T) {
	tempDir := t.TempDir()

	f1 := filepath.Join(tempDir, "20260914020000_second.sql")
	f2 := filepath.Join(tempDir, "20260914010000_first.sql")

	_ = os.WriteFile(f1, []byte("-- +goks Up\nCREATE TABLE second(id INT);\n-- +goks Down\nDROP TABLE second;"), 0644)
	_ = os.WriteFile(f2, []byte("-- +goks Up\nCREATE TABLE first(id INT);\n-- +goks Down\nDROP TABLE first;"), 0644)

	migs, err := orm.LoadMigrationsFromDir(tempDir)
	if err != nil {
		t.Fatalf("LoadMigrationsFromDir failed: %v", err)
	}

	if len(migs) != 2 {
		t.Fatalf("expected 2 migrations, got %d", len(migs))
	}
	// Verify sorted ascending by version
	if migs[0].Version != "20260914010000" || migs[1].Version != "20260914020000" {
		t.Errorf("expected sorted migrations, got: %v, %v", migs[0].Version, migs[1].Version)
	}
}

func TestMigrate_And_Status(t *testing.T) {
	testMockDrv.Reset()

	db, err := orm.Connect("mock_sqlite", "testdb")
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer db.Close()

	migrations := []orm.Migration{
		{
			Version: "20260914010000",
			Name:    "create_users",
			Up:      "CREATE TABLE users (id INTEGER PRIMARY KEY);",
			Down:    "DROP TABLE users;",
		},
	}

	// 1. Check initial status
	statuses, err := orm.GetStatus(db, migrations)
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}
	if len(statuses) != 1 || statuses[0].Applied {
		t.Fatalf("expected 1 pending migration, got %+v", statuses)
	}

	// 2. Run Migrate
	if err := orm.Migrate(db, migrations); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	// Verify query log contains CREATE TABLE and INSERT INTO goks_migrations
	testMockDrv.mu.Lock()
	var foundUp, foundInsert bool
	for _, q := range testMockDrv.queries {
		if strings.Contains(q, "CREATE TABLE users") {
			foundUp = true
		}
		if strings.Contains(q, "INSERT INTO goks_migrations") {
			foundInsert = true
		}
	}
	testMockDrv.mu.Unlock()

	if !foundUp || !foundInsert {
		t.Fatalf("expected Up SQL and tracking record in queries, got: %v", testMockDrv.queries)
	}
}

func TestMigrate_FailureRollback(t *testing.T) {
	testMockDrv.Reset()
	testMockDrv.mu.Lock()
	testMockDrv.failing = true
	testMockDrv.mu.Unlock()

	db, err := orm.Connect("mock_sqlite", "testdb")
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer db.Close()

	failingMigration := []orm.Migration{
		{
			Version: "20260914030000",
			Name:    "failing_step",
			Up:      "INVALID SQL STATEMENT HERE;",
			Down:    "DROP TABLE temp;",
		},
	}

	err = orm.Migrate(db, failingMigration)
	if err == nil {
		t.Fatal("expected migration error on invalid statement, got nil")
	}
}

func TestRunSeeders(t *testing.T) {
	testMockDrv.Reset()

	db, err := orm.Connect("mock_sqlite", "testdb")
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer db.Close()

	tempDir := t.TempDir()
	seedSQL := "INSERT INTO roles (id, name) VALUES (1, 'Admin');"
	_ = os.WriteFile(filepath.Join(tempDir, "01_roles.sql"), []byte(seedSQL), 0644)

	executed, err := orm.RunSeeders(db, tempDir)
	if err != nil {
		t.Fatalf("RunSeeders failed: %v", err)
	}
	if len(executed) != 1 || executed[0] != "01_roles.sql" {
		t.Fatalf("unexpected executed seeders: %v", executed)
	}
}
