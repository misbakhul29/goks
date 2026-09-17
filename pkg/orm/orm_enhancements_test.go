package orm_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/misbakhul29/goks/pkg/orm"
)

type CustomCategory struct {
	orm.Model
	Name string `db:"name"`
}

func (c *CustomCategory) TableName() string {
	return "categories"
}

func TestORM_TableNamer(t *testing.T) {
	dir := t.TempDir()
	db, err := orm.Connect("sqlite", filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer db.Close()

	_, err = db.Raw().Exec(`
		CREATE TABLE categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	cat := &CustomCategory{Name: "Electronics"}
	if err := orm.Create(db, cat); err != nil {
		t.Fatalf("expected create into 'categories' to succeed: %v", err)
	}

	found, err := orm.Query[CustomCategory](db).Where("name = ?", "Electronics").First()
	if err != nil {
		t.Fatalf("expected to find category: %v", err)
	}
	if found.Name != "Electronics" {
		t.Fatalf("expected Electronics, got %q", found.Name)
	}
}

func TestORM_WithContextCancellation(t *testing.T) {
	dir := t.TempDir()
	db, err := orm.Connect("sqlite", filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = orm.Query[CustomCategory](db).WithContext(ctx).Find()
	if err == nil {
		t.Fatal("expected error due to cancelled context")
	}
}

func TestORM_NilDatabaseSafety(t *testing.T) {
	// Call on empty/unconnected query builder
	var nilDB *orm.Database
	b := orm.Query[CustomCategory](nilDB)

	_, err := b.Find()
	if err == nil {
		t.Fatal("expected error for uninitialized DB on Find()")
	}

	_, err = b.First()
	if err == nil {
		t.Fatal("expected error for uninitialized DB on First()")
	}

	_, err = b.Count()
	if err == nil {
		t.Fatal("expected error for uninitialized DB on Count()")
	}

	cat := &CustomCategory{Name: "Fail"}
	if err := orm.Create(nilDB, cat); err == nil {
		t.Fatal("expected error for uninitialized DB on Create()")
	}
}

type StandaloneModel struct {
	ID        uint       `db:"id"`
	Title     string     `db:"title"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at"`
}

func (s *StandaloneModel) TableName() string {
	return "standalones"
}

func TestORM_StandaloneIDModel(t *testing.T) {
	dir := t.TempDir()
	db, err := orm.Connect("sqlite", filepath.Join(dir, "standalone.db"))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer db.Close()

	_, err = db.Raw().Exec(`
		CREATE TABLE standalones (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// 1. Create
	m := &StandaloneModel{Title: "Initial Title"}
	if err := orm.Create(db, m); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if m.ID == 0 {
		t.Fatal("expected non-zero ID after Create")
	}

	// 2. Update
	m.Title = "Updated Title"
	if err := orm.Update(db, m); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	found, err := orm.FindByID[StandaloneModel](db, m.ID)
	if err != nil {
		t.Fatalf("find failed: %v", err)
	}
	if found.Title != "Updated Title" {
		t.Fatalf("expected Updated Title, got %q", found.Title)
	}

	// 3. Soft Delete
	if err := orm.Delete(db, m); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if m.DeletedAt == nil {
		t.Fatal("expected DeletedAt to be set on model after Delete")
	}

	// Should not be found without WithTrashed
	_, err = orm.Query[StandaloneModel](db).Where("id = ?", m.ID).First()
	if err == nil {
		t.Fatal("expected soft-deleted record to be hidden from default query")
	}

	// 4. Restore
	if err := orm.Restore(db, m); err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	if m.DeletedAt != nil {
		t.Fatal("expected DeletedAt to be nil after Restore")
	}

	restored, err := orm.FindByID[StandaloneModel](db, m.ID)
	if err != nil {
		t.Fatalf("find after restore failed: %v", err)
	}
	if restored.ID != m.ID {
		t.Fatalf("expected ID %d, got %d", m.ID, restored.ID)
	}

	// 5. Hard Delete
	if err := orm.HardDelete(db, m); err != nil {
		t.Fatalf("hard delete failed: %v", err)
	}

	// Should be completely gone even with WithTrashed
	_, err = orm.Query[StandaloneModel](db).WithTrashed().Where("id = ?", m.ID).First()
	if err == nil {
		t.Fatal("expected hard-deleted record to be permanently gone")
	}
}
