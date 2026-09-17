package orm_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/misbakhul29/goks/pkg/orm"
)

type Account struct {
	orm.Model
	Username string `db:"username"`
	Password string `db:"password"`
	Status   string `db:"status"`
	HookLog  string `db:"-"`
}

func (a *Account) BeforeCreate() error {
	if a.Username == "forbidden" {
		return errors.New("cannot create forbidden user")
	}
	if !strings.HasPrefix(a.Password, "hashed:") {
		a.Password = "hashed:" + a.Password
	}
	return nil
}

func (a *Account) AfterCreate(ctx context.Context) error {
	a.HookLog = "created_with_ctx"
	return nil
}

func (a *Account) BeforeUpdate() error {
	if a.Status == "locked" {
		return errors.New("cannot update locked account")
	}
	return nil
}

func (a *Account) AfterUpdate() error {
	a.HookLog = "updated"
	return nil
}

func (a *Account) BeforeDelete(ctx context.Context) error {
	if a.Username == "admin" {
		return errors.New("cannot delete admin account")
	}
	return nil
}

func (a *Account) AfterDelete() error {
	a.HookLog = "deleted"
	return nil
}

func (a *Account) AfterFind() error {
	a.HookLog = "loaded:" + a.Username
	return nil
}

func setupHookDB(t *testing.T) *orm.Database {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "hook_test.db")
	db, err := orm.Connect("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to connect db: %v", err)
	}

	_, err = db.Raw().Exec(`
		CREATE TABLE accounts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL,
			password TEXT NOT NULL,
			status TEXT NOT NULL,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		);
	`)
	if err != nil {
		t.Fatalf("failed to create accounts table: %v", err)
	}
	return db
}

func TestORM_Hooks_Create(t *testing.T) {
	db := setupHookDB(t)
	defer db.Close()

	// 1. Success case: password gets hashed in BeforeCreate, HookLog set in AfterCreate
	acc := &Account{
		Username: "alice",
		Password: "secretpassword",
		Status:   "active",
	}

	err := orm.Create(db, acc)
	if err != nil {
		t.Fatalf("unexpected error creating account: %v", err)
	}
	if acc.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if acc.Password != "hashed:secretpassword" {
		t.Fatalf("expected password to be hashed in BeforeCreate, got %q", acc.Password)
	}
	if acc.HookLog != "created_with_ctx" {
		t.Fatalf("expected HookLog to be set by AfterCreate, got %q", acc.HookLog)
	}

	// 2. Failure case: BeforeCreate returns error, aborting insertion
	badAcc := &Account{
		Username: "forbidden",
		Password: "plain",
		Status:   "active",
	}
	err = orm.Create(db, badAcc)
	if err == nil {
		t.Fatal("expected error from BeforeCreate hook, got nil")
	}
	if !strings.Contains(err.Error(), "cannot create forbidden user") {
		t.Fatalf("expected error message to contain hook error, got %v", err)
	}
	if badAcc.ID != 0 {
		t.Fatalf("expected ID to be 0 for aborted creation, got %d", badAcc.ID)
	}
}

func TestORM_Hooks_Update(t *testing.T) {
	db := setupHookDB(t)
	defer db.Close()

	acc := &Account{
		Username: "bob",
		Password: "pass",
		Status:   "active",
	}
	if err := orm.Create(db, acc); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// 1. Successful update
	acc.Password = "newpass"
	if err := orm.Update(db, acc); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if acc.HookLog != "updated" {
		t.Fatalf("expected HookLog 'updated', got %q", acc.HookLog)
	}

	// 2. Aborted update via BeforeUpdate hook
	acc.Status = "locked"
	err := orm.Update(db, acc)
	if err == nil {
		t.Fatal("expected error updating locked account, got nil")
	}
	if !strings.Contains(err.Error(), "cannot update locked account") {
		t.Fatalf("expected hook error, got %v", err)
	}
}

func TestORM_Hooks_Delete(t *testing.T) {
	db := setupHookDB(t)
	defer db.Close()

	admin := &Account{Username: "admin", Password: "adm", Status: "active"}
	if err := orm.Create(db, admin); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Attempt delete admin
	err := orm.Delete(db, admin)
	if err == nil {
		t.Fatal("expected error deleting admin account, got nil")
	}
	if !strings.Contains(err.Error(), "cannot delete admin account") {
		t.Fatalf("expected hook error, got %v", err)
	}

	// Delete normal user
	user := &Account{Username: "charlie", Password: "pwd", Status: "active"}
	if err := orm.Create(db, user); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if err := orm.Delete(db, user); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if user.HookLog != "deleted" {
		t.Fatalf("expected HookLog 'deleted', got %q", user.HookLog)
	}
}

func TestORM_Hooks_AfterFind(t *testing.T) {
	db := setupHookDB(t)
	defer db.Close()

	acc := &Account{Username: "dave", Password: "pwd", Status: "active"}
	if err := orm.Create(db, acc); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// 1. Find via Query builder
	list, err := orm.Query[Account](db).Where("username = ?", "dave").Find()
	if err != nil {
		t.Fatalf("find failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 result, got %d", len(list))
	}
	if list[0].HookLog != "loaded:dave" {
		t.Fatalf("expected HookLog 'loaded:dave', got %q", list[0].HookLog)
	}

	// 2. FindByID
	found, err := orm.FindByID[Account](db, acc.ID)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if found.HookLog != "loaded:dave" {
		t.Fatalf("expected HookLog 'loaded:dave', got %q", found.HookLog)
	}
}

func TestORM_Context_Timeout(t *testing.T) {
	db := setupHookDB(t)
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Immediately cancelled

	acc := &Account{Username: "eva", Password: "pwd", Status: "active"}
	err := orm.CreateContext(ctx, db, acc)
	if err == nil {
		t.Fatal("expected error due to cancelled context, got nil")
	}

	_, err = orm.FindByIDContext[Account](ctx, db, 1)
	if err == nil {
		t.Fatal("expected error due to cancelled context on FindByIDContext, got nil")
	}
}

func TestORM_TransactionContext(t *testing.T) {
	db := setupHookDB(t)
	defer db.Close()

	ctx := context.Background()
	err := db.TransactionContext(ctx, func(tx *sql.Tx) error {
		_, err := tx.Exec("INSERT INTO accounts (username, password, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
			"txuser", "txpass", "active", time.Now(), time.Now())
		return err
	})
	if err != nil {
		t.Fatalf("transaction failed: %v", err)
	}

	found, err := orm.Query[Account](db).Where("username = ?", "txuser").First()
	if err != nil {
		t.Fatalf("expected txuser to be committed, got err: %v", err)
	}
	if found.Username != "txuser" {
		t.Fatalf("expected txuser, got %s", found.Username)
	}
}
