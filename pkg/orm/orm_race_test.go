package orm

// Concurrent transaction and query race tests for pkg/orm.
// Run with: go test -race ./pkg/orm/...
//
// These tests use SQLite :memory: (one db per test) to avoid shared state.
// They cover the scenarios flagged in the architecture audit as untested:
//   - concurrent Transaction() calls
//   - panic inside fn rolls back, not commit
//   - concurrent reads and writes (Builder.Find + Create)

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

// createRaceDB returns a fresh in-memory SQLite database with the
// testitems table already created. Each test must use its own db to
// avoid cross-test state sharing.
func createRaceDB(t *testing.T) *Database {
	t.Helper()
	db, err := OpenSQLite(":memory:")
	if err != nil {
		t.Fatalf("OpenSQLite failed: %v", err)
	}
	_, err = db.Raw().Exec(`CREATE TABLE testitems (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		title      TEXT,
		price      INTEGER,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`)
	if err != nil {
		t.Fatalf("create table failed: %v", err)
	}
	return db
}

// TestTransaction_ConcurrentCommits verifies that N goroutines each
// running their own Transaction() can all commit without data loss or panic.
func TestTransaction_ConcurrentCommits(t *testing.T) {
	db := createRaceDB(t)
	defer db.Close()

	const goroutines = 10
	var wg sync.WaitGroup
	var errCount atomic.Int32

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		title := fmt.Sprintf("item-%d", i)
		go func() {
			defer wg.Done()
			err := db.Transaction(context.Background(), func(tx *sql.Tx) error {
				_, err := tx.Exec(
					"INSERT INTO testitems (title, price, created_at, updated_at) VALUES (?, ?, datetime('now'), datetime('now'))",
					title, i,
				)
				return err
			})
			if err != nil {
				errCount.Add(1)
			}
		}()
	}
	wg.Wait()

	if errCount.Load() != 0 {
		t.Fatalf("%d transactions failed (expected 0)", errCount.Load())
	}

	var count int
	if err := db.Raw().QueryRow("SELECT COUNT(*) FROM testitems").Scan(&count); err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != goroutines {
		t.Fatalf("expected %d rows after concurrent inserts, got %d", goroutines, count)
	}
}

// TestTransaction_PanicRollback verifies that a panic inside fn is
// re-raised (not swallowed) and the transaction is rolled back.
func TestTransaction_PanicRollback(t *testing.T) {
	db := createRaceDB(t)
	defer db.Close()

	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		_ = db.Transaction(context.Background(), func(tx *sql.Tx) error {
			_, _ = tx.Exec(
				"INSERT INTO testitems (title, price, created_at, updated_at) VALUES (?, ?, datetime('now'), datetime('now'))",
				"should-rollback", 0,
			)
			panic("intentional panic in transaction")
		})
	}()

	if !panicked {
		t.Fatal("expected panic to propagate out of Transaction()")
	}

	var count int
	if err := db.Raw().QueryRow("SELECT COUNT(*) FROM testitems").Scan(&count); err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 rows after panic rollback, got %d", count)
	}
}

// TestTransaction_NilContextDefaultsToBackground verifies that passing
// nil ctx does not panic and the transaction completes normally.
func TestTransaction_NilContextDefaultsToBackground(t *testing.T) {
	db := createRaceDB(t)
	defer db.Close()

	//nolint:staticcheck // intentional nil to test the nil-ctx guard in Transaction()
	err := db.Transaction(nil, func(tx *sql.Tx) error {
		_, err := tx.Exec(
			"INSERT INTO testitems (title, price, created_at, updated_at) VALUES (?, ?, datetime('now'), datetime('now'))",
			"nil-ctx", 1,
		)
		return err
	})
	if err != nil {
		t.Fatalf("transaction with nil ctx failed: %v", err)
	}
}

// TestTransaction_ConcurrentReadWrite exercises the race detector on
// concurrent Create (write) and Builder.Find (read) against the same db.
func TestTransaction_ConcurrentReadWrite(t *testing.T) {
	db := createRaceDB(t)
	defer db.Close()

	// Pre-insert one row so Find always has something to return.
	if err := Create(db, &TestItem{Title: "seed", Price: 0}); err != nil {
		t.Fatalf("seed insert failed: %v", err)
	}

	const goroutines = 20
	var wg sync.WaitGroup
	var readErr, writeErr atomic.Int32

	for i := 0; i < goroutines; i++ {
		wg.Add(2)

		// Writer
		title := fmt.Sprintf("concurrent-%d", i)
		go func() {
			defer wg.Done()
			if err := Create(db, &TestItem{Title: title, Price: i}); err != nil {
				writeErr.Add(1)
			}
		}()

		// Reader
		go func() {
			defer wg.Done()
			if _, err := Query[TestItem](db).Find(); err != nil {
				readErr.Add(1)
			}
		}()
	}
	wg.Wait()

	if readErr.Load() != 0 || writeErr.Load() != 0 {
		t.Fatalf("concurrent read/write errors: reads=%d writes=%d",
			readErr.Load(), writeErr.Load())
	}
}
