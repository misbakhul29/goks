package orm

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

type TestItem struct {
	Model
	Title string `db:"title"`
	Price int    `db:"price"`
}

func TestBuilder_OrderBy_Validation(t *testing.T) {
	b := Query[TestItem]()

	// Safe inputs should succeed
	b.OrderBy("id DESC")
	b.OrderBy("title ASC")
	b.OrderBy("created_at")
	b.OrderBy("t.id, t.title DESC")

	// Dangerous inputs should panic
	unsafeInputs := []string{
		"id; DROP TABLE users;--",
		"id UNION SELECT * FROM passwords",
		"1=1",
		"(SELECT password FROM secrets)",
	}

	for _, input := range unsafeInputs {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Fatalf("expected panic for unsafe OrderBy %q", input)
				}
			}()
			Query[TestItem]().OrderBy(input)
		}()
	}
}

func TestFieldPointers_EmbeddedModel(t *testing.T) {
	item := &TestItem{}
	cols := []string{"id", "created_at", "title", "price", "unknown_col"}

	ptrs := fieldPointers(item, cols)
	if len(ptrs) != len(cols) {
		t.Fatalf("expected %d pointers, got %d", len(cols), len(ptrs))
	}

	// Verify that the pointers actually point to the embedded Model fields
	// and top level fields
	idPtr, ok := ptrs[0].(*uint)
	if !ok || idPtr == nil {
		t.Fatalf("expected *uint for id, got %T", ptrs[0])
	}
	*idPtr = 42
	if item.ID != 42 {
		t.Fatalf("expected item.ID to be 42, got %d", item.ID)
	}

	titlePtr, ok := ptrs[2].(*string)
	if !ok || titlePtr == nil {
		t.Fatalf("expected *string for title, got %T", ptrs[2])
	}
	*titlePtr = "Test Product"
	if item.Title != "Test Product" {
		t.Fatalf("expected item.Title to be 'Test Product', got %s", item.Title)
	}
}

func TestDialect_SQLite(t *testing.T) {
	d := dialectFor("sqlite3")
	if d.Placeholder(1) != "?" {
		t.Errorf("expected ? for SQLite placeholder, got %s", d.Placeholder(1))
	}
	if d.SupportsReturning() {
		t.Errorf("expected SupportsReturning to be false for SQLite")
	}

	d2 := dialectFor("sqlite")
	if d2.Placeholder(1) != "?" {
		t.Errorf("expected ? for SQLite placeholder, got %s", d2.Placeholder(1))
	}
}

func TestOpenSQLite_RegistersDefaultDriver(t *testing.T) {
	db, err := OpenSQLite(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("OpenSQLite() error = %v", err)
	}
	defer db.Close()

	if err := db.Raw().Ping(); err != nil {
		t.Fatalf("SQLite ping error = %v", err)
	}
}

func TestDatabase_Transaction_Commit(t *testing.T) {
	db, err := OpenSQLite(":memory:")
	if err != nil {
		t.Fatalf("OpenSQLite failed: %v", err)
	}
	defer db.Close()

	_, err = db.Raw().Exec("CREATE TABLE test_tx (id INTEGER PRIMARY KEY, val TEXT)")
	if err != nil {
		t.Fatalf("create table failed: %v", err)
	}

	err = db.Transaction(context.Background(), func(tx *sql.Tx) error {
		_, err := tx.Exec("INSERT INTO test_tx (val) VALUES (?)", "committed")
		return err
	})
	if err != nil {
		t.Fatalf("transaction failed: %v", err)
	}

	var val string
	err = db.Raw().QueryRow("SELECT val FROM test_tx WHERE id = 1").Scan(&val)
	if err != nil || val != "committed" {
		t.Fatalf("expected 'committed', got val=%q, err=%v", val, err)
	}
}

func TestDatabase_Transaction_Rollback(t *testing.T) {
	db, err := OpenSQLite(":memory:")
	if err != nil {
		t.Fatalf("OpenSQLite failed: %v", err)
	}
	defer db.Close()

	_, err = db.Raw().Exec("CREATE TABLE test_tx (id INTEGER PRIMARY KEY, val TEXT)")
	if err != nil {
		t.Fatalf("create table failed: %v", err)
	}

	err = db.Transaction(context.Background(), func(tx *sql.Tx) error {
		_, _ = tx.Exec("INSERT INTO test_tx (val) VALUES (?)", "should_rollback")
		return sql.ErrConnDone // deliberate error
	})
	if err == nil {
		t.Fatal("expected error from failed transaction")
	}

	var count int
	_ = db.Raw().QueryRow("SELECT COUNT(*) FROM test_tx").Scan(&count)
	if count != 0 {
		t.Fatalf("expected 0 rows after rollback, got %d", count)
	}
}

func TestORM_CRUD_SoftDelete_Restore(t *testing.T) {
	db, err := OpenSQLite(":memory:")
	if err != nil {
		t.Fatalf("OpenSQLite failed: %v", err)
	}
	defer db.Close()

	_, err = db.Raw().Exec(`CREATE TABLE testitems (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT,
		price INTEGER,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`)
	if err != nil {
		t.Fatalf("failed to create testitems table: %v", err)
	}

	// 1. Create
	item := &TestItem{Title: "Laptop", Price: 1500}
	if err := Create(db, item); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if item.ID == 0 {
		t.Fatal("expected non-zero ID after Create")
	}

	// 2. Query Find
	items, err := Query[TestItem](db).Find()
	if err != nil || len(items) != 1 {
		t.Fatalf("expected 1 item, got %d (err: %v)", len(items), err)
	}

	// 3. Soft Delete
	if err := Delete(db, item); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if item.DeletedAt == nil {
		t.Fatal("expected item.DeletedAt to be populated after soft delete")
	}

	// 4. Default query should NOT find it
	itemsAfterDelete, err := Query[TestItem](db).Find()
	if err != nil {
		t.Fatalf("Query after delete failed: %v", err)
	}
	if len(itemsAfterDelete) != 0 {
		t.Fatalf("expected 0 items in normal query, got %d", len(itemsAfterDelete))
	}

	// 5. WithTrashed query SHOULD find it
	trashedItems, err := Query[TestItem](db).WithTrashed().Find()
	if err != nil || len(trashedItems) != 1 {
		t.Fatalf("expected 1 item with WithTrashed, got %d (err: %v)", len(trashedItems), err)
	}

	// 6. Restore
	if err := Restore(db, item); err != nil {
		t.Fatalf("Restore failed: %v", err)
	}
	if item.DeletedAt != nil {
		t.Fatal("expected item.DeletedAt to be nil after Restore")
	}

	// 7. Normal query now finds it again
	restoredItems, err := Query[TestItem](db).Find()
	if err != nil || len(restoredItems) != 1 {
		t.Fatalf("expected 1 item after Restore, got %d (err: %v)", len(restoredItems), err)
	}
}
