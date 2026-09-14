package generator_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/misbakhul29/goks/internal/compiler"
	"github.com/misbakhul29/goks/internal/generator"
)

func TestGenerateRouter_FileBasedAPIRoutes(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Create go.mod
	goModContent := `module example.com/testapp

go 1.22
`
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// 2. Create app/page.go
	appDir := filepath.Join(tempDir, "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("failed to create app dir: %v", err)
	}
	pageContent := `package app

import "github.com/misbakhul29/goks/pkg/component"

type Page struct{ component.ComponentBase }
func (p *Page) Render() *component.Node { return component.H("div", nil, "Home") }
`
	if err := os.WriteFile(filepath.Join(appDir, "page.go"), []byte(pageContent), 0644); err != nil {
		t.Fatalf("failed to write page.go: %v", err)
	}

	// 3. Create app/api/hello/route.go with GET & POST
	apiHelloDir := filepath.Join(appDir, "api", "hello")
	if err := os.MkdirAll(apiHelloDir, 0755); err != nil {
		t.Fatalf("failed to create api/hello dir: %v", err)
	}
	helloRouteContent := `package hello

import "github.com/misbakhul29/goks/pkg/router"

func GET(c *router.Context) error {
	return c.JSON(map[string]string{"message": "hello"})
}

func POST(c *router.Context) error {
	return c.Status(201).JSON(map[string]string{"status": "created"})
}

func helperFunc() string {
	return "internal"
}
`
	if err := os.WriteFile(filepath.Join(apiHelloDir, "route.go"), []byte(helloRouteContent), 0644); err != nil {
		t.Fatalf("failed to write api/hello/route.go: %v", err)
	}

	// 4. Create app/api/users/[id]/route.go with GET & DELETE (testing Next.js-style bracket params)
	apiUsersIdDir := filepath.Join(appDir, "api", "users", "[id]")
	if err := os.MkdirAll(apiUsersIdDir, 0755); err != nil {
		t.Fatalf("failed to create api/users/[id] dir: %v", err)
	}
	usersRouteContent := `package id

import "github.com/misbakhul29/goks/pkg/router"

func GET(c *router.Context) error {
	id := c.Param("id")
	return c.JSON(map[string]string{"user_id": id})
}

func DELETE(c *router.Context) error {
	return c.Status(204).Text("")
}
`
	if err := os.WriteFile(filepath.Join(apiUsersIdDir, "route.go"), []byte(usersRouteContent), 0644); err != nil {
		t.Fatalf("failed to write api/users/[id]/route.go: %v", err)
	}

	// 5. Test compiler.PrepareWorkspace normalizes bracketed folder to _id
	if err := compiler.PrepareWorkspace(tempDir); err != nil {
		t.Fatalf("PrepareWorkspace failed: %v", err)
	}
	normalizedWorkspaceFile := filepath.Join(tempDir, ".goks", "workspace", "app", "api", "users", "_id", "route.go")
	if _, err := os.Stat(normalizedWorkspaceFile); err != nil {
		t.Fatalf("expected normalized workspace file at %s, got: %v", normalizedWorkspaceFile, err)
	}

	// 6. Run GenerateRouter
	if err := generator.GenerateRouter(tempDir, false); err != nil {
		t.Fatalf("GenerateRouter failed: %v", err)
	}

	// 7. Verify .goks/entry/server_main.go
	serverMainPath := filepath.Join(tempDir, ".goks", "entry", "server_main.go")
	serverMainBytes, err := os.ReadFile(serverMainPath)
	if err != nil {
		t.Fatalf("failed to read server_main.go: %v", err)
	}
	serverMain := string(serverMainBytes)

	// Check imports in server_main.go
	if !strings.Contains(serverMain, "example.com/testapp/app/api/hello") {
		t.Errorf("expected server_main.go to import api/hello, got:\n%s", serverMain)
	}
	if !strings.Contains(serverMain, "example.com/testapp/app/api/users/_id") {
		t.Errorf("expected server_main.go to import normalized api/users/_id, got:\n%s", serverMain)
	}

	// Check route registrations in server_main.go
	expectedRegistrations := []string{
		`srv.Router().Handle("GET", "/api/hello",`,
		`srv.Router().Handle("POST", "/api/hello",`,
		`srv.Router().Handle("GET", "/api/users/:id",`,
		`srv.Router().Handle("DELETE", "/api/users/:id",`,
	}
	for _, expected := range expectedRegistrations {
		if !strings.Contains(serverMain, expected) {
			t.Errorf("server_main.go missing expected registration: %s\nFull content:\n%s", expected, serverMain)
		}
	}

	// 8. Security screening verification:
	// Client WASM (router.go) MUST NOT import or bundle route.go!
	routerPath := filepath.Join(tempDir, ".goks", "entry", "router.go")
	routerBytes, err := os.ReadFile(routerPath)
	if err != nil {
		t.Fatalf("failed to read router.go: %v", err)
	}
	routerSrc := string(routerBytes)
	if strings.Contains(routerSrc, "api/hello") || strings.Contains(routerSrc, "api/users") {
		t.Fatalf("SECURITY VIOLATION: router.go (client WASM) leaks backend API route packages:\n%s", routerSrc)
	}
}
