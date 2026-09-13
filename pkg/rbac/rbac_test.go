package rbac_test

import (
	"sync"
	"testing"

	"github.com/misbakhul29/goks/pkg/auth"
	"github.com/misbakhul29/goks/pkg/rbac"
)

func TestRegistry_DefineAndHas(t *testing.T) {
	reg := rbac.NewRegistry()
	reg.Define("editor", "posts.create", "posts.edit")
	reg.Define("admin", "*")

	if !reg.Has("editor", "posts.create") {
		t.Errorf("expected editor to have posts.create")
	}
	if reg.Has("editor", "posts.delete") {
		t.Errorf("expected editor to not have posts.delete")
	}
	if !reg.Has("admin", "posts.delete") {
		t.Errorf("expected admin with wildcard to have posts.delete")
	}
	if reg.Has("unknown_role", "posts.create") {
		t.Errorf("expected non-existent role to return false")
	}
}

func TestRegistry_UserHas_NilUser(t *testing.T) {
	reg := rbac.NewRegistry()
	reg.Define("admin", "*")

	// Must not panic on nil user
	if reg.UserHas(nil, "any.perm") {
		t.Errorf("expected nil user to return false")
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	reg := rbac.NewRegistry()
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func(idx int) {
			defer wg.Done()
			reg.Define("role", "perm1", "perm2")
		}(i)
		go func() {
			defer wg.Done()
			user := &auth.User{Roles: []string{"role"}}
			_ = reg.UserHas(user, "perm1")
			_ = reg.Has("role", "perm2")
		}()
	}
	wg.Wait()
}
