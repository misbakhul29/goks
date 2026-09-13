package orm

import (
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
