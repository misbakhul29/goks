# GoKS (Go Kickstart)

GoKS is a modern, full-stack Go web framework designed for building full-stack web applications entirely in Go—from server-side routing and APIs to client-side interactive UI via WebAssembly. **No JavaScript or Node.js required.**

It embraces a "batteries-included" philosophy, combining a React/Next.js-like developer experience with the performance, type safety, and simplicity of Go: **GOX (JSX-like syntax in Go)**, **Tailwind CSS v4**, file-system routing, built-in ORM, auth, WebSockets, and hot-reloading CLI.

---

## ✨ Features

- **🚀 GOX Syntax (`.gox`):** Write declarative, JSX-like markup directly in Go (`<div><h1>...</h1></div>`, `<c.Button />`, `{variable}`). Transpiled automatically into clean Go code in an isolated workspace.
- **⚡ Tailwind CSS v4 Built-in:** Zero-config Tailwind CSS v4 powered by the standalone Rust compiler. No Node.js or npm dependencies required—GoKS downloads and manages the CLI automatically.
- **🎨 `goks ui` Component System:** A `shadcn/ui`-inspired CLI generator that adds production-ready, fully styled Tailwind `.gox` components (`button`, `input`, `card`, `dialog`, `badge`, `dropdown`, `table`) right into your project.
- **⚡ TinyGo WASM Support:** Optional `--compiler=tinygo` build mode yielding ultra-compact client WebAssembly bundles (<300 KB) with instant boot times.
- **🏝️ Islands Architecture & Selective Hydration:** Static pages render 100% pure HTML with 0-WASM overhead. WebAssembly hydration is selectively loaded only for interactive client islands (`component.ClientBase` or event handlers).
- **🔄 Progressive Server Actions (`pkg/action`):** Next.js-style server functions with dual-mode execution—submits seamlessly via async WASM `fetch()`, with automatic zero-JS HTML form fallback and CSRF protection.
- **🗄️ Built-in ORM with SQLite:** Fluent, type-safe query builder (`orm.Query[T]().Where(...)`), auto-timestamps, soft-deletes, migrations, and zero-config local database with `orm.OpenSQLite("app.db")`.
- **🪝 React-like Hooks:** Functional component state management with `component.UseState()` and dynamic node wrapping with `component.Any()`.
- **🛠️ Powerful CLI Tooling:**
  - `goks new` — Scaffold full-stack applications with an opinionated structure.
  - `goks dev` — Instant hot-reload server with WASM compilation, Tailwind watcher, and browser compile-error overlay.
  - `goks page` & `goks generate` — Scaffold pages, reusable components, and ORM models.
  - `goks ui add` — Add accessible, styled components into `components/ui/`.
  - `goks build` & `goks start` — Optimized production builds and server launcher.
  - `goks build --standalone` — Bundle everything (WASM, CSS, assets) into a **single self-contained binary**. No project source needed at runtime.
- **📁 File-System Routing:** Next.js App Router style routing inside the `app/` directory (`page.gox`, `layout.gox`).
- **🔒 Authentication & RBAC:** Zero-dependency JWT, session management, and Role-Based Access Control middleware.
- **🔌 WebSockets:** Hub pattern, typed event routing (`hub.On("event")`), and room management.
- **📦 Global State Store:** Reactive global store (`store.New()`) for WASM client state synchronization.
- **🛡️ Production Middleware Stack:** Built-in `Logger`, `CORS`, `Secure`, `RequestID`, `Compress`, `MaxBytes`, and `Timeout`.

---

## 📦 Installation

Install the `goks` CLI using `go install`:

```bash
go install github.com/misbakhul29/goks@latest
```

> **Note:** Ensure your `$(go env GOPATH)/bin` is added to your system's `$PATH`.

---

## 🚀 Quick Start

### 1. Create a new app
```bash
goks new myapp
cd myapp
go mod tidy
```

### 2. Start the development server
```bash
goks dev
```

Your app will be running at `http://localhost:3000` with live reloading.
- Tailwind CSS v4 compiles automatically in the background.
- If you have a compilation error, GoKS displays an in-browser error overlay and recovers as soon as you save the fix.

### 3. Generate pages and components
```bash
# Generate a new page (app/about/page.gox)
goks page about
# or
goks generate page blog/detail

# Generate a reusable component (app/components/card.gox)
goks generate component Card

# Generate a database model (models/post.go)
goks generate model Post
```

### 4. Build for production

**Standard build** — outputs server binary + assets:
```bash
goks build
goks start 3000
```

**Standalone build** — one binary, run anywhere (like Next.js `output: 'standalone'`):
```bash
goks build --standalone
# Output: .goks/standalone/server

# Run it anywhere:
./.goks/standalone/server
PORT=8080 ./.goks/standalone/server
```

---

## 💻 GOX (`.gox`) Syntax

GoKS introduces `.gox` files, allowing you to write HTML/JSX-like tags directly inside Go functions and methods while maintaining 100% type safety.

### Example: Page Component (`app/page.gox`)

```go
package app

import (
	"github.com/misbakhul29/goks/pkg/component"
	c "myapp/app/components"
)

type Page struct {
	component.ComponentBase
}

func (p *Page) Render() *component.Node {
	title := "Welcome to GoKS"

	return (
		<main class="min-h-screen bg-slate-50 dark:bg-slate-900 p-8">
			<div class="max-w-4xl mx-auto">
				<h1 class="text-4xl font-bold text-slate-900 dark:text-white mb-4">
					{title}
				</h1>
				<p class="text-slate-600 dark:text-slate-400 mb-6">
					Fullstack web application running purely on Go and WebAssembly.
				</p>
				<c.Button Label="Get Started" Variant="primary" />
			</div>
		</main>
	)
}
```

### Example: State & Hooks (`app/components/counter.gox`)

GoKS supports React-like hooks such as `UseState`:

```go
package components

import (
	"fmt"
	"github.com/misbakhul29/goks/pkg/component"
)

type Counter struct {
	component.ComponentBase
	Initial int
}

func (c *Counter) Render() *component.Node {
	count, setCount := component.UseState(c.Initial)

	return (
		<div class="flex items-center gap-4 p-4 border rounded-xl">
			<span class="text-xl font-semibold">Count: {count}</span>
			<button 
				class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
				onClick={func() { setCount(count + 1) }}>
				Increment
			</button>
		</div>
	)
}
```

> **VS Code Tip:** Add the following to your `.vscode/settings.json` to get syntax highlighting for `.gox` files:
> ```json
> {
>   "files.associations": {
>     "*.gox": "go"
>   }
> }
> ```

---

## 🗂️ Project Structure

A scaffolded GoKS app follows an organized, modular structure:

```text
myapp/
├── app/                  # File-system routes & UI components (.gox)
│   ├── layout.gox        # Root layout wrapper (HTML shell & navigation)
│   ├── page.gox          # Home route (/)
│   ├── about/
│   │   └── page.gox      # Route (/about)
│   └── components/       # App-specific UI components (.gox)
│       └── hero.gox
├── api/                  # Backend REST API routes and handlers
│   └── routes.go
├── config/               # App & server configuration
│   └── goks.config.go
├── middleware/           # Custom HTTP middlewares
│   └── logger.go
├── models/               # Database ORM models
│   └── user.go
├── repositories/         # Database access layer
├── services/             # Application business logic
├── public/               # Static assets & Tailwind CSS entrypoint
│   ├── global.css        # Tailwind v4 configuration (@import "tailwindcss";)
│   └── favicon.ico
├── .goks/                # GoKS build cache & isolated workspace (auto-managed)
├── go.mod
└── go.sum
```

---

## 📖 Backend & Fullstack Features

### 1. Database ORM (`models/user.go`)

GoKS includes a built-in query builder with soft-delete and timestamp support:

```go
package models

import "github.com/misbakhul29/goks/pkg/orm"

type User struct {
	orm.Model // Provides ID, CreatedAt, UpdatedAt, DeletedAt
	Name  string `db:"name"`
	Email string `db:"email"`
}

// Find a single record:
// user, err := orm.Query[models.User]().Where("email = $1", email).First()

// Query list with pagination:
// users, err := orm.Query[models.User]().
//     Where("deleted_at IS NULL").
//     OrderBy("created_at DESC").
//     Limit(10).
//     Find()

// Insert new record:
// orm.Create(orm.DB, &User{Name: "Alex", Email: "alex@example.com"})
```

### 2. REST API & Middleware (`api/routes.go`)

Register API routes and attach built-in JWT / RBAC middlewares:

```go
package api

import (
	"github.com/misbakhul29/goks/pkg/router"
	"github.com/misbakhul29/goks/pkg/auth"
)

func RegisterRoutes(r *router.Router) {
	// Public endpoint
	r.GET("/api/health", func(ctx *router.Context) error {
		return ctx.JSON(map[string]string{"status": "ok"})
	})

	// Protected endpoint with JWT & Role-Based Access Control
	r.GET("/api/admin/users", AdminUsersHandler, auth.JWTMiddleware(), auth.RequireRole("admin"))
}
```

### 3. `goks ui` Component System (shadcn/ui for GoKS)

Quickly scaffold beautiful, accessible, production-ready Tailwind UI components directly in `.gox`:

```bash
# List available components: button, input, card, dialog, badge, dropdown, table
goks ui list

# Add components to components/ui/
goks ui add button input card dialog

# Or install all available components at once
goks ui add all
```

Use them seamlessly inside your `.gox` pages:

```go
package page

import "myapp/components/ui"

func (p *MyPage) Render() *component.Node {
	return <div class="p-8 space-y-4">
		<ui.Card>
			<ui.CardHeader>
				<ui.CardTitle>Welcome back</ui.CardTitle>
				<ui.CardDescription>Enter your credentials to continue</ui.CardDescription>
			</ui.CardHeader>
			<ui.CardContent>
				<ui.Input placeholder="Email address" type="email" />
			</ui.CardContent>
			<ui.CardFooter>
				<ui.Button variant="primary">Sign In</ui.Button>
			</ui.CardFooter>
		</ui.Card>
	</div>
}
```

### 4. Progressive Server Actions (`pkg/action`)

Execute backend Go code directly from HTML forms or client events with automatic zero-JS progressive enhancement:

```go
package actions

import (
	"github.com/misbakhul29/goks/pkg/action"
)

func init() {
	action.Register("subscribeNewsletter", func(ctx *action.Context) (any, error) {
		email := ctx.FormData.Get("email")
		// Save to database...
		return map[string]string{"message": "Subscribed successfully!"}, nil
	})
}
```

In your `.gox` markup:
```html
<form action={action.URL("subscribeNewsletter")} method="POST">
	<input type="email" name="email" required />
	<button type="submit">Subscribe</button>
</form>
```
- **WASM Client**: Automatically intercepts submission with `fetch()`, updates state without page reloading.
- **No-JS / SSR Fallback**: Submits as a regular HTML POST and redirects back cleanly with HTTP 303.
- **Built-in Security**: Enforces same-origin CSRF verification on all actions.

### 5. Islands Architecture & Selective Hydration

GoKS defaults to zero-WASM pure HTML for static pages, eliminating client runtime overhead:

- **Static Pages**: If a page contains no interactive handlers, GoKS serves **pure HTML** with `0 KB` WASM downloaded.
- **Interactive Islands**: When interactive events (`onClick`, `onChange`) or components embedding `component.ClientBase` are present, GoKS automatically hydrates the WebAssembly runtime.

```go
type CounterIsland struct {
	component.ComponentBase
	component.ClientBase // Marks this component as an interactive client island
}
```

### 6. Ultra-Compact WASM with TinyGo (`--compiler=tinygo`)

For production deployments where bundle size matters, compile with TinyGo to reduce WASM payload from ~2.5 MB down to **< 300 KB**:

```bash
# Development with TinyGo
goks dev --compiler=tinygo

# Production build with TinyGo
goks build --compiler=tinygo

# Standalone single-binary with embedded TinyGo WASM
goks build --compiler=tinygo --standalone
```

### 7. Zero-Config Embedded SQLite (`pkg/orm`)

Get started instantly without spinning up external database servers:

```go
import "github.com/misbakhul29/goks/pkg/orm"

func main() {
	// Automatically initializes SQLite database at app.db
	db, err := orm.OpenSQLite("app.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Use standard fluent query builder
	users, _ := orm.Query[User]().Where("active", "=", true).Find(db)
}
```

---

## ⌨️ CLI Reference

| Command | Description |
| :--- | :--- |
| `goks new <app>` | Create a new GoKS application |
| `goks dev [-p port] [--compiler=tinygo]` | Start development server with hot-reload (default: port 3000) |
| `goks page <route>` | Scaffold a new `.gox` page in `app/<route>/page.gox` |
| `goks generate page <route>` | Alias to generate a page |
| `goks generate component <Name>` | Scaffold a reusable component in `app/components/<name>.gox` |
| `goks generate model <Name>` | Scaffold an ORM model in `models/<name>.go` |
| `goks ui list` | List all available UI components |
| `goks ui add <name... \| all>` | Add styled Tailwind components to `components/ui/` |
| `goks build [--compiler=tinygo]` | Compile WASM bundle, Tailwind CSS, and production server binary |
| `goks build --standalone` | Build a **single self-contained binary** with all assets embedded |
| `goks start [port]` | Run production server (built inside `.goks/build/`) |
| `goks version` | Display current GoKS version |

---

## 🚢 Deployment

### Standard (`goks build`)
```bash
goks build
# Serve from the project directory:
goks start 3000
# or with custom PORT:
PORT=8080 goks start
```

### Standalone (`goks build --standalone`) ✨

Produces a **single binary** at `.goks/standalone/server` with WASM, CSS, and all static files embedded inside — no project source required at runtime.

```bash
goks build --standalone
```

**VPS / Bare metal:**
```bash
scp .goks/standalone/server user@host:/opt/myapp/server
ssh user@host 'chmod +x /opt/myapp/server && PORT=80 /opt/myapp/server'
```

**Docker (ultra-minimal image):**
```dockerfile
FROM scratch
COPY .goks/standalone/server /server
EXPOSE 3000
CMD ["/server"]
```

**Railway / Render / Fly.io:**
Just upload `.goks/standalone/server` and set `PORT` as an environment variable. No build step needed on the platform.

**GitHub Actions → Vercel/Server:**
```yaml
- name: Build standalone
  run: |
    go install github.com/misbakhul29/goks@latest
    goks build --standalone

- name: Deploy
  run: scp .goks/standalone/server ${{ secrets.SERVER_USER }}@${{ secrets.SERVER_HOST }}:/opt/app/server
```

---

## 🤝 Contributing

Contributions, issues, and feature requests are welcome! Feel free to check out the issues or submit a pull request.

## 📄 License

This project is licensed under the [MIT License](LICENSE).

