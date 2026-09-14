package compatibility_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/misbakhul29/goks/pkg/action"
	"github.com/misbakhul29/goks/pkg/auth"
	"github.com/misbakhul29/goks/pkg/component"
	"github.com/misbakhul29/goks/pkg/metadata"
	"github.com/misbakhul29/goks/pkg/orm"
	"github.com/misbakhul29/goks/pkg/rbac"
	"github.com/misbakhul29/goks/pkg/router"
	"github.com/misbakhul29/goks/pkg/store"
)

type TestUserModel struct {
	orm.Model
	Name  string `json:"name"`
	Email string `json:"email"`
}

// TestStableAPICompileContract verifies at compile time that stable public APIs
// in pkg/* retain their expected exported types, signatures, and contracts.
func TestStableAPICompileContract(t *testing.T) {
	// 1. pkg/router contract
	r := router.New()
	var _ http.Handler = r
	r.GET("/test", func(c *router.Context) error {
		_ = c.Param("id")
		_ = c.Query("q")
		return c.Status(http.StatusOK).JSON(map[string]string{"ok": "true"})
	})
	group := r.Group("/api", func(next router.Handler) router.Handler {
		return func(c *router.Context) error {
			return next(c)
		}
	})
	if group == nil {
		t.Fatal("expected non-nil router group")
	}

	// 2. pkg/action contract
	action.Register("test.action", func(ctx *action.Context) (any, error) {
		return map[string]string{"status": "ok"}, nil
	})
	var _ http.Handler = action.Handler()
	_ = action.URL("test.action")
	_ = action.RegisteredActions()

	// 3. pkg/component contract
	el := component.H("div", component.Props{"class": "container"}, component.Text("hello"), component.Fragment())
	if el == nil {
		t.Fatal("expected non-nil element")
	}
	rendered := component.RenderToString(el)
	if rendered == "" {
		t.Fatal("expected rendered HTML")
	}
	suspense := component.Suspense(component.SuspenseProps{
		Fallback: component.Text("loading"),
		Async: func(ctx context.Context) (*component.Node, error) {
			return component.Text("resolved"), nil
		},
		Timeout: 5 * time.Second,
	})
	if suspense == nil {
		t.Fatal("expected suspense node")
	}

	// 4. pkg/orm contract
	db, err := orm.OpenSQLite(":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite in-memory: %v", err)
	}
	defer db.Close()

	q := orm.Query[TestUserModel](db).
		Where("name = ?", "alice").
		Limit(1).
		Offset(0).
		OrderBy("id DESC")
	if q == nil {
		t.Fatal("expected non-nil query builder")
	}

	// 5. pkg/auth contract
	mgr := auth.New(24 * time.Hour)
	if mgr == nil {
		t.Fatal("expected non-nil session manager")
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	sess, err := mgr.Login(w, req, &auth.User{ID: 123, Name: "alice", Email: "alice@example.com", Roles: []string{"admin"}})
	if err != nil || sess == nil {
		t.Fatalf("failed to login: %v", err)
	}
	mgr.Logout(w, req)

	jwtMgr := auth.NewJWT("super-secret-key-that-is-at-least-32-bytes-long", 24*time.Hour)
	if jwtMgr == nil {
		t.Fatal("expected non-nil jwt manager")
	}
	var _ router.MiddlewareFunc = auth.JWTMiddleware()
	var _ router.MiddlewareFunc = auth.Required()

	// 6. pkg/rbac contract
	reg := rbac.NewRegistry()
	reg.Define("admin", rbac.Permission("posts:write"))
	if !reg.UserHas(&auth.User{Roles: []string{"admin"}}, rbac.Permission("posts:write")) {
		t.Fatal("expected admin to have posts:write permission")
	}
	var _ router.MiddlewareFunc = rbac.Can(rbac.Permission("posts:write"))
	var _ router.MiddlewareFunc = rbac.HasRole("admin")

	// 7. pkg/metadata contract
	m := metadata.Metadata{
		Title:       "Test Page",
		Description: "A test description",
	}
	metaHTML := metadata.RenderHTML(m)
	if metaHTML == "" {
		t.Fatal("expected non-empty rendered metadata")
	}

	// 8. pkg/store contract
	st := store.New("initial")
	if st.Get() != "initial" {
		t.Fatal("expected store initial value")
	}
	unsub := st.Subscribe(func(s string) {})
	unsub()
}
