// Package rbac provides Role-Based Access Control for GoKS.
package rbac

import (
	"net/http"

	"github.com/misbakhulmunir/goks/pkg/auth"
	"github.com/misbakhulmunir/goks/pkg/router"
)

// Permission represents a named ability (e.g. "posts.create").
type Permission string

// Role defines a set of allowed permissions.
type Role struct {
	Name        string
	Permissions []Permission
}

// Registry holds all defined roles.
type Registry struct {
	roles map[string]*Role
}

// Default is the global RBAC registry.
var Default = NewRegistry()

// NewRegistry creates a new RBAC registry.
func NewRegistry() *Registry {
	return &Registry{roles: make(map[string]*Role)}
}

// Define registers a role with a set of permissions.
//
//	rbac.Default.Define("admin", "posts.create", "posts.delete", "users.manage")
//	rbac.Default.Define("editor", "posts.create", "posts.update")
func (r *Registry) Define(roleName string, perms ...Permission) {
	r.roles[roleName] = &Role{Name: roleName, Permissions: perms}
}

// Has checks if a role has a specific permission.
func (r *Registry) Has(roleName string, perm Permission) bool {
	role, ok := r.roles[roleName]
	if !ok {
		return false
	}
	for _, p := range role.Permissions {
		if p == perm || p == "*" {
			return true
		}
	}
	return false
}

// UserHas checks if the current user has a permission (checks all their roles).
func (r *Registry) UserHas(user *auth.User, perm Permission) bool {
	for _, roleName := range user.Roles {
		if r.Has(roleName, perm) {
			return true
		}
	}
	return false
}

// -----------------------------------------------------------------------
// Middleware
// -----------------------------------------------------------------------

// HasRole is middleware that requires the current user to have one of the given roles.
func HasRole(roles ...string) router.MiddlewareFunc {
	return func(next router.Handler) router.Handler {
		return func(ctx *router.Context) error {
			user := auth.CurrentUser(ctx.Request())
			if user == nil {
				return ctx.Status(http.StatusUnauthorized).JSON(map[string]string{
					"error": "authentication required",
				})
			}
			for _, required := range roles {
				for _, userRole := range user.Roles {
					if userRole == required {
						return next(ctx)
					}
				}
			}
			return ctx.Status(http.StatusForbidden).JSON(map[string]string{
				"error": "insufficient permissions",
			})
		}
	}
}

// Can is middleware that requires the current user to have a specific permission.
func Can(perm Permission) router.MiddlewareFunc {
	return func(next router.Handler) router.Handler {
		return func(ctx *router.Context) error {
			user := auth.CurrentUser(ctx.Request())
			if user == nil {
				return ctx.Status(http.StatusUnauthorized).JSON(map[string]string{
					"error": "authentication required",
				})
			}
			if !Default.UserHas(user, perm) {
				return ctx.Status(http.StatusForbidden).JSON(map[string]string{
					"error": "permission denied: " + string(perm),
				})
			}
			return next(ctx)
		}
	}
}
