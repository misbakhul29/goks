# GoKS (Go Kickstart)

GoKS is a modern, full-stack Go framework designed for building web applications entirely in Go—from server-side routing and APIs to client-side UI via WebAssembly. No JavaScript is required.

It embraces a "batteries-included" philosophy, providing everything needed to build robust web apps quickly: a seamless developer experience, built-in ORM, authentication, WebSocket support, and a powerful CLI.

## Features

- **CLI Tooling:** `goks new`, `goks dev`, and `goks build` for scaffolding, hot-reloading (with an intelligent error overlay), and production builds.
- **File-System Routing:** Organize your routes in the `app/` directory similar to the Next.js App Router.
- **Go-Native Virtual DOM:** Build your UI components using Go code (no HTML templates) with a chainable builder API (`html.Div()`, `html.H1()`, etc.).
- **SSR & WASM:** Server-Side Rendering out of the box, with seamless handoff to WebAssembly on the client.
- **Built-in ORM:** Fluent query builder (`orm.Query[T]().Where(...)`), auto-timestamps, soft-deletes, and migrations for PostgreSQL, MySQL, and SQLite.
- **Authentication & RBAC:** Session management, zero-dependency JWT, and Role-Based Access Control middleware.
- **WebSockets:** Hub pattern, typed event routing (`hub.On("event")`), and room management.
- **Global State Management:** Reactive global store (`store.New()`) for your WASM apps.
- **Rich Middleware Stack:** Built-in `Logger`, `CORS`, `Secure`, `RequestID`, `Compress`, `MaxBytes`, and `Timeout`.

## Installation

To install the `goks` CLI tool, run:

```bash
go install github.com/misbakhulmunir/goks/cli/cmd/goks@latest
```

*(Note: Ensure your `$(go env GOPATH)/bin` is in your system's `$PATH`)*

## Quick Start

Create a new GoKS application:

```bash
goks new myapp
cd myapp
go mod tidy
```

Start the development server with hot-reloading:

```bash
goks dev
```

Your app will be running at `http://localhost:3000`. 
The dev server includes a smart error overlay; if you introduce a compile error in your Go code, the browser will display the error directly and auto-recover when fixed.

## Project Structure

A scaffolded GoKS app comes with an opinionated structure:

- `app/` - Your UI components and file-system routes (`page.go`, `layout.go`).
- `api/` - REST API handlers and routes (`routes.go`).
- `components/` - Reusable UI components.
- `models/` - Database models and schemas.
- `repositories/` - Data access layer (DB queries).
- `services/` - Business logic and use cases.
- `middleware/` - Custom HTTP middlewares.
- `public/` - Static assets (CSS, images).
- `config/` - App configuration (e.g., `goks.config.go`).

## Examples

### 1. UI Components (`app/page.go`)

Build your UI entirely in Go:

```go
package app

import (
	"github.com/misbakhulmunir/goks/pkg/component"
	"github.com/misbakhulmunir/goks/pkg/html"
)

type Page struct {
	component.ComponentBase
}

func (p *Page) Render() *component.Node {
	return html.Main(
		html.H1("Hello, GoKS!").Class("text-4xl font-bold"),
		html.P("Welcome to my application."),
	).Class("container mx-auto p-8")
}
```

### 2. ORM (`models/user.go`)

Interact with your database using the built-in ORM:

```go
package models

import "github.com/misbakhulmunir/goks/pkg/orm"

type User struct {
	orm.Model // Adds ID, CreatedAt, UpdatedAt, DeletedAt
	Name  string ` + "`" + `db:"name"` + "`" + `
	Email string ` + "`" + `db:"email"` + "`" + `
}

// Find a user:
// user, _ := orm.Query[models.User]().Where("email = $1", email).First()

// Create a user:
// orm.Create(orm.DB, &User{Name: "Budi", Email: "budi@example.com"})
```

### 3. API & Middleware (`api/routes.go`)

Set up your REST API with built-in middlewares:

```go
package api

import (
	"net/http"
	"github.com/misbakhulmunir/goks/pkg/router"
	"github.com/misbakhulmunir/goks/pkg/auth"
)

func RegisterRoutes(r *router.Router) {
	r.GET("/api/public", func(ctx *router.Context) error {
		return ctx.JSON(map[string]string{"status": "ok"})
	})

	// Protected route with JWT and RBAC
	r.GET("/api/admin", AdminHandler, auth.JWTMiddleware(), auth.RequireRole("admin"))
}
```

## Contributing

Contributions are welcome. Feel free to open issues or submit pull requests.

## License

MIT License. See [LICENSE](LICENSE) for details.
