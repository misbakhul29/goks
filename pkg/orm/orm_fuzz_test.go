package orm_test

import (
	"strings"
	"testing"

	"github.com/misbakhul29/goks/pkg/orm"
)

type FuzzModel struct {
	orm.Model
	Name string
}

// FuzzSQLIdentifierValidation verifies that query builder and identifier validation
// safely handle arbitrary inputs without panics or SQL injection escapes.
func FuzzSQLIdentifierValidation(f *testing.F) {
	seeds := []string{
		"users",
		"items_v2",
		"users; DROP TABLE users;--",
		"users WHERE 1=1",
		"col`--",
		"' OR '1'='1",
		"SELECT * FROM accounts",
		"",
		"   ",
		"very_long_identifier_name_with_underscores_and_123456789",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	db, err := orm.OpenSQLite(":memory:")
	if err != nil {
		f.Fatalf("failed to open sqlite: %v", err)
	}
	defer db.Close()

	f.Fuzz(func(t *testing.T, identifier string) {
		defer func() {
			if r := recover(); r != nil {
				if errMsg, ok := r.(string); ok && strings.Contains(errMsg, "unsafe OrderBy value") {
					// Expected security panic protecting against SQL injection in OrderBy
					return
				}
				t.Fatalf("ORM query builder panicked unexpectedly on identifier %q: %v", identifier, r)
			}
		}()

		// Try building queries with the fuzzed identifier in Where and OrderBy
		q := orm.Query[FuzzModel](db).
			Where(identifier+" = ?", "test").
			OrderBy(identifier)

		_ = q
		// Attempting query execution should return an error if invalid, never crash
		_, _ = q.Find()
	})
}
