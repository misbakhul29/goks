package cli

// Project scaffold templates.
// Kept in a dedicated file so new.go stays focused on logic.

var tmplGoMod = `module {{.Module}}

go 1.22

require github.com/misbakhul29/goks {{.Version}}
`

var tmplLayout = `package app

import (
	"github.com/misbakhul29/goks/pkg/component"
	"github.com/misbakhul29/goks/pkg/html"
)

// Layout acts as the root template (equivalent to layout.tsx)
type Layout struct {
	component.ComponentBase
	Children component.Renderable
}

func (l *Layout) Render() *component.Node {
	return (
		<div class="antialiased">
			{l.Children}
		</div>
	)
}
`

var tmplPage = `package app

import (
	"github.com/misbakhul29/goks/pkg/component"
	"github.com/misbakhul29/goks/pkg/html"
	c "{{.Module}}/app/components"
)

// Page is the home page component (equivalent to page.tsx).
type Page struct {
	component.ComponentBase
}

func (p *Page) Render() *component.Node {
	return (
		<main class="min-h-screen bg-white dark:bg-slate-900">
			<c.Hero Title="Welcome to {{.AppName}}" Subtitle="Built with GoKS + Tailwind CSS v4" />
		</main>
	)
}
`

var tmplHeroComponent = `package components

import (
	"github.com/misbakhul29/goks/pkg/component"
	"github.com/misbakhul29/goks/pkg/html"
)

// Hero is a reusable hero section component.
type Hero struct {
	component.ComponentBase
	Title    string
	Subtitle string
}

func (h *Hero) Render() *component.Node {
	return (
		<div class="flex flex-col items-center justify-center min-h-screen text-slate-900 dark:text-slate-50 p-8 transition-colors duration-300">
			<h1 class="text-5xl font-bold tracking-tight mb-4">{h.Title}</h1>
			<p class="text-xl text-slate-600 dark:text-slate-400 mb-10 max-w-2xl text-center">{h.Subtitle}</p>
			<div class="flex gap-4">
				<button class="px-5 py-2.5 bg-slate-900 dark:bg-slate-50 text-white dark:text-slate-900 font-medium rounded-lg hover:bg-slate-800 dark:hover:bg-slate-200 transition-colors">Get Started</button>
				<button class="px-5 py-2.5 bg-transparent border border-slate-300 dark:border-slate-700 font-medium rounded-lg hover:bg-slate-50 dark:hover:bg-slate-800 transition-colors">Documentation</button>
			</div>
		</div>
	)
}
`

var tmplGlobalCss = `@import "tailwindcss";

/* 
  Global CSS for GoKS 
  Tailwind v4 features lightning-fast Rust engine and zero-config by default.
*/
`

var tmplConfig = `// goks.config.go — GoKS project configuration.
package config

import (
	"{{.Module}}/middleware"
	"github.com/misbakhul29/goks/runtime/server"
	"github.com/misbakhul29/goks/pkg/router"
)

// ServerConfig returns the global server configuration.
func ServerConfig() server.Config {
	return server.Config{
		Middlewares: []router.MiddlewareFunc{
			// Built-in middlewares — uncomment to enable:
			// router.Secure(),             // Security headers (XSS, Clickjacking, etc.)
			// router.RequestID(),          // Inject X-Request-Id into every request
			// router.Compress(),           // Gzip compression for supported clients
			// router.MaxBytes(1 << 20),    // Limit request body to 1MB
			middleware.Logger(),
		},
	}
}
`

var tmplMiddleware = `package middleware

import (
	"log"
	"time"

	"github.com/misbakhul29/goks/pkg/router"
)

// Logger is a sample global middleware that logs request execution time.
//
// To create your own middleware, return a router.MiddlewareFunc:
//
//	func MyMiddleware() router.MiddlewareFunc {
//	    return func(next router.Handler) router.Handler {
//	        return func(ctx *router.Context) error {
//	            // do something before
//	            err := next(ctx)
//	            // do something after
//	            return err
//	        }
//	    }
//	}
func Logger() router.MiddlewareFunc {
	return func(next router.Handler) router.Handler {
		return func(ctx *router.Context) error {
			start := time.Now()

			// Call the next handler in the chain
			err := next(ctx)

			// Log the request
			log.Printf("[App] %s %s took %v", ctx.Request().Method, ctx.Request().URL.Path, time.Since(start))

			return err
		}
	}
}
`

var tmplGitignore = `.goks/
dist/
*.wasm
vendor/
.env
`

var tmplReadme = `# {{.AppName}}

Built with [GoKS](https://github.com/misbakhul29/goks) — the fullstack Go framework.

## Getting started

` + "```bash\n" + `go mod tidy
goks dev       # Start dev server at http://localhost:3000
goks build     # Build for production
` + "```"

// -----------------------------------------------------------------------
// Example files for models/, services/, repositories/, api/, components/
// -----------------------------------------------------------------------

var tmplExampleModel = `package models

import "github.com/misbakhul29/goks/pkg/orm"

// User is an example database model.
// Embed orm.Model to get ID, CreatedAt, UpdatedAt, and soft-delete (DeletedAt) for free.
//
// Usage:
//
//	// Connect to DB in your server config:
//	orm.Connect("postgres", "host=localhost user=postgres dbname=myapp sslmode=disable")
//
//	// Create a new user:
//	user := &models.User{Name: "Budi", Email: "budi@example.com"}
//	orm.Create(orm.DB, user) // INSERT INTO users ... RETURNING id
//
//	// Query:
//	users, _ := orm.Query[models.User]().Where("email = $1", "budi@example.com").Find()
//	user, _  := orm.Query[models.User]().Where("id = $1", 1).First()
//
//	// Update:
//	user.Name = "Budi Santoso"
//	orm.Save(orm.DB, user) // UPDATE users SET ...
//
//	// Soft-delete (sets deleted_at, not actually removed):
//	orm.Delete(orm.DB, user)
//
//	// Hard delete:
//	orm.HardDelete(orm.DB, user)
type User struct {
	orm.Model
	Name  string ` + "`db:\"name\"`" + `
	Email string ` + "`db:\"email\"`" + `
}
`

var tmplExampleService = `package services

// UserService handles business logic for users.
// Keep business rules here — not in handlers or repositories.
//
// Dependency injection pattern:
//
//	svc := services.NewUserService()
//	err := svc.Register("Budi", "budi@example.com")
type UserService struct {
	// Inject dependencies here:
	// repo *repositories.UserRepository
	// jwt  *auth.JWT
}

func NewUserService() *UserService {
	return &UserService{}
}

// Register creates a new user account.
// Add your validation, hashing, and business logic here.
func (s *UserService) Register(name, email string) error {
	// Example:
	// if name == "" || email == "" {
	//     return errors.New("name and email are required")
	// }
	// user := &models.User{Name: name, Email: email}
	// return orm.Create(orm.DB, user)
	return nil
}
`

var tmplExampleRepository = `package repositories

// UserRepository handles all database access for User models.
// Services should call repositories — never touch orm.DB directly from services.
//
// GoKS ORM query examples:
//
//	// Find all users:
//	users, _ := orm.Query[models.User]().Find()
//
//	// Find by condition:
//	user, _ := orm.Query[models.User]().Where("email = $1", email).First()
//
//	// Paginate:
//	users, _ := orm.Query[models.User]().OrderBy("created_at DESC").Limit(10).Offset(20).Find()
//
//	// Count:
//	total, _ := orm.Query[models.User]().Where("deleted_at IS NULL").Count()
type UserRepository struct{}

func NewUserRepository() *UserRepository {
	return &UserRepository{}
}
`

var tmplExampleAPI = `package api

import (
	"net/http"

	"github.com/misbakhul29/goks/pkg/router"
)

// RegisterRoutes wires up all REST API routes onto the given router.
// Call this from your server config or main entry point.
//
// Example in config/goks.config.go:
//
//	func ServerConfig() server.Config {
//	    return server.Config{
//	        Routes: api.RegisterRoutes,
//	        ...
//	    }
//	}
func RegisterRoutes(r *router.Router) {
	// Health check
	r.GET("/api/health", func(ctx *router.Context) error {
		return ctx.JSON(map[string]string{"status": "ok"})
	})

	// Protected routes with JWT:
	// r.GET("/api/me", MeHandler, auth.JWTMiddleware())
	//
	// Role-based access:
	// r.DELETE("/api/users/:id", DeleteUser, auth.JWTMiddleware(), auth.RequireRole("admin"))
}

// ExampleHandler demonstrates how to write a GoKS API handler.
func ExampleHandler(ctx *router.Context) error {
	// Read path param:   ctx.Param("id")
	// Read query string: ctx.Query("page")
	// Read header:       ctx.Header("Authorization")
	// Bind JSON body:    ctx.Bind(&myStruct{})
	// Send JSON:         ctx.JSON(data)
	// Send text:         ctx.Text("hello")
	// Send HTML:         ctx.HTML("<h1>hi</h1>")
	// Redirect:          ctx.Redirect("/login")
	// Set status:        ctx.Status(http.StatusCreated).JSON(data)

	return ctx.Status(http.StatusOK).JSON(map[string]string{
		"message": "Hello from GoKS API!",
	})
}
`

var tmplExampleComponent = `package components

import (
	"github.com/misbakhul29/goks/pkg/component"
	"github.com/misbakhul29/goks/pkg/html"
)

// Button is a reusable UI component example.
// Props are simply struct fields. Pass them via component.C() in a page:
//
//	<components.Button Label="Click me" Variant="primary" />
//
// You can also use html.* directly without wrapping in a component:
//
//	<button class="px-4 py-2 bg-blue-600 text-white rounded">Click me</button>
type Button struct {
	component.ComponentBase
	Label   string
	Variant string // "primary" | "secondary"
}

func (b *Button) Render() *component.Node {
	class := "px-4 py-2 rounded-lg font-medium transition-colors"
	if b.Variant == "primary" {
		class += " bg-blue-600 text-white hover:bg-blue-700"
	} else {
		class += " border border-slate-300 hover:bg-slate-50"
	}
	return (
		<button class="{class}">{b.Label}</button>
	)
}
`
