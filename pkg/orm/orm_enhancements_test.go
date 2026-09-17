package orm_test

import (
	"context"
	"path/filepath"
	"testing"

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
