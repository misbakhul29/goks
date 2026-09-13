# GoKS (Go Kickstart)

<div align="center">

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://golang.org)
[![Release](https://img.shields.io/badge/release-v0.8.1-6366F1?style=for-the-badge&logo=github)](https://github.com/misbakhul29/goks/releases)
[![License](https://img.shields.io/badge/license-MIT-10B981?style=for-the-badge)](LICENSE)
[![WASM](https://img.shields.io/badge/WebAssembly-Enabled-654FF0?style=for-the-badge&logo=webassembly&logoColor=white)](https://webassembly.org)
[![Tailwind](https://img.shields.io/badge/Tailwind_CSS-v4-06B6D4?style=for-the-badge&logo=tailwindcss&logoColor=white)](https://tailwindcss.com)

**The Modern Fullstack Go Web Framework.**  
*Backend APIs + WebAssembly Frontend, 100% Type-Safe Go. Zero Node.js or JavaScript Required.*

[Getting Started](#-quick-start) • [Features](#-features) • [Why GoKS?](#-why-goks) • [UI Components](#-goks-ui-component-system) • [Server Actions](#-progressive-server-actions-pkgaction) • [Deployment](#-deployment-guide)

</div>

---

## ⚡ Why GoKS?

GoKS bridges the gap between modern React/Next.js developer ergonomics and the legendary performance, type-safety, and single-binary deployment of Go.

| Feature | GoKS | Next.js / Remix | templ + HTMX | Fiber / Gin |
| :--- | :---: | :---: | :---: | :---: |
| **Language Stack** | **100% Go** (Fullstack) | TypeScript / JS | Go + HTML Attributes | Go (Backend Only) |
| **Frontend Runtime** | **WebAssembly (Go)** | JS Virtual DOM | Browser Native / DOM swap | None / Raw Templates |
| **Node.js / npm Required?** | **❌ No (Zero npm)** | ✅ Required | ❌ No | ❌ No |
| **Server Actions** | **✅ Built-in (`pkg/action`)** | ✅ Yes | ⚠️ Partial (HTMX triggers) | ❌ Manual endpoints |
| **Selective Hydration** | **✅ Islands (0-WASM static)** | ⚠️ Partial (React RSC) | ❌ N/A | ❌ N/A |
| **Component Generator** | **✅ `goks ui` (shadcn-style)** | ✅ `shadcn/ui` | ❌ None | ❌ None |
| **Tailwind CSS v4** | **✅ Built-in standalone** | ⚠️ Needs Node.js/npm | ⚠️ External CLI setup | ⚠️ External setup |
| **Production Artifact** | **Single Standalone Binary** | Node runtime + node_modules | Single Binary | Single Binary |

---

## ✨ Features

- **🚀 GOX Syntax (`.gox`):** Declarative, JSX-like markup directly inside Go (`<div><h1>{title}</h1></div>`, `<ui.Button />`). Compiled directly into Go code in an isolated workspace.
- **🎨 `goks ui` Component System:** Built-in CLI generator inspired by `shadcn/ui`. Add accessible, styled Tailwind components (`button`, `input`, `card`, `dialog`, `badge`, `dropdown`, `table`) right into `components/ui/`.
- **🏝️ Islands Architecture & Selective Hydration:** Static pages render **100% pure HTML with 0 KB WASM**. WebAssembly hydration is selectively loaded only for pages with client interactivity (`component.ClientBase` or event handlers).
- **⚡ TinyGo WASM Support (`--compiler=tinygo`):** Compile client WebAssembly bundles down to **< 300 KB** for blazing-fast mobile and edge load times.
- **🔄 Progressive Server Actions (`pkg/action`):** Next.js-style server functions with dual-mode execution: async `fetch()` via WASM in modern browsers, with automatic zero-JS HTML form fallback and built-in CSRF protection.
- **⚡ Tailwind CSS v4 Built-in:** Zero-config Tailwind CSS v4 powered by the standalone Rust compiler. No Node.js or npm dependencies required—GoKS manages everything automatically.
- **🗄️ Built-in ORM & SQLite:** Fluent, type-safe query builder (`orm.Query[T]().Where(...)`), auto-timestamps, soft-deletes, migrations, and instant zero-config local development with `orm.OpenSQLite("app.db")`.
- **🪝 React-like Hooks:** Functional component state management with `component.UseState()` and dynamic child wrapping with `component.Any()`.
- **🛠️ Powerful CLI Tooling:**
  - `goks new` — Scaffold full-stack apps with opinionated structure.
  - `goks dev` — Hot-reloading dev server with WASM compilation, Tailwind watcher, and browser compile-error overlay.
  - `goks ui add` — Add styled components into `components/ui/`.
  - `goks page` & `goks generate` — Scaffold routes, reusable components, and ORM models.
  - `goks build --standalone` — Produce a **single self-contained binary** embedding all assets (WASM, CSS, HTML).
- **📁 File-System Routing:** Next.js App Router style file-based routing inside `app/` (`page.gox`, `layout.gox`).
- **🔒 Authentication & RBAC:** Zero-dependency JWT tokens, session management, and Role-Based Access Control middleware.
- **🔌 Realtime WebSockets:** WebSocket hub, typed event routing (`hub.On("chat", handler)`), and room broadcasting.
- **📦 Reactive Global Store:** `store.New()` for cross-component reactive state synchronization in client WASM.
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

### 2. Start development server
```bash
goks dev
```
Your app is live at `http://localhost:3000` with instant hot-reloading!
- Tailwind CSS v4 compiles automatically in the background.
- If code fails to compile, GoKS displays an in-browser error overlay and auto-recovers on file save.

### 3. Add UI components (`goks ui`)
```bash
# Add essential components
goks ui add button input card dialog

# Or install all available components:
goks ui add all
```

### 4. Build for production

**Standard build:**
```bash
goks build
goks start 3000
```

**Single-binary Standalone build (Run anywhere, no source needed):**
```bash
goks build --standalone
./.goks/standalone/server
```

**Ultra-compact WASM build (< 300 KB with TinyGo):**
```bash
goks build --compiler=tinygo --standalone
```

---

## 💻 GOX (`.gox`) Syntax Guide

GOX allows you to write HTML/JSX-like markup directly in Go while retaining strict compile-time type safety.

### Example: Page Component (`app/page.gox`)

```go
package app

import (
	"github.com/misbakhul29/goks/pkg/component"
	"myapp/components/ui"
)

type Page struct {
	component.ComponentBase
}

func (p *Page) Render() *component.Node {
	title := "Welcome to GoKS"

	return (
		<main class="min-h-screen bg-slate-50 dark:bg-slate-900 p-8">
			<div class="max-w-4xl mx-auto space-y-6">
				<h1 class="text-4xl font-bold text-slate-900 dark:text-white">
					{title}
				</h1>
				<p class="text-slate-600 dark:text-slate-400">
					Fullstack web application powered purely by Go and WebAssembly.
				</p>
				<ui.Card>
					<ui.CardHeader>
						<ui.CardTitle>Quick Actions</ui.CardTitle>
						<ui.CardDescription>Get started right away</ui.CardDescription>
					</ui.CardHeader>
					<ui.CardContent>
						<ui.Button variant="primary">Get Started</ui.Button>
					</ui.CardContent>
				</ui.Card>
			</div>
		</main>
	)
}
```

### Example: Client Interactive Island & State (`app/components/counter.gox`)

To mark a component as an interactive client island that hydrates WebAssembly, embed `component.ClientBase` or use event handlers:

```go
package components

import (
	"github.com/misbakhul29/goks/pkg/component"
	"myapp/components/ui"
)

type Counter struct {
	component.ComponentBase
	component.ClientBase // Designates this component as an interactive WASM island
	Initial int
}

func (c *Counter) Render() *component.Node {
	count, setCount := component.UseState(c.Initial)

	return (
		<div class="flex items-center gap-4 p-4 border rounded-xl bg-white shadow-sm">
			<span class="text-xl font-semibold">Count: {count}</span>
			<ui.Button 
				variant="primary"
				onClick={func() { setCount(count + 1) }}>
				Increment
			</ui.Button>
		</div>
	)
}
```

> **VS Code Tip:** Add this to `.vscode/settings.json` for syntax highlighting:
> ```json
> {
>   "files.associations": {
>     "*.gox": "go"
>   }
> }
> ```

---

## 🎨 `goks ui` Component System

GoKS includes a first-class CLI component generator inspired by `shadcn/ui`. Components are written in pure `.gox` with Tailwind CSS v4 styling, fully customizable directly inside your repository.

```bash
# List all available components
goks ui list

# Install individual components
goks ui add button input card dialog badge table dropdown

# Or install everything at once
goks ui add all
```

| Component | File | Description & Variants |
| :--- | :--- | :--- |
| `button` | `components/ui/button.gox` | Variants (`primary`, `secondary`, `destructive`, `outline`, `ghost`), sizes (`sm`, `md`, `lg`) |
| `input` | `components/ui/input.gox` | Styled text, email, password input with focus rings, disabled and error states |
| `card` | `components/ui/card.gox` | Modular card container: `Card`, `CardHeader`, `CardTitle`, `CardDescription`, `CardContent`, `CardFooter` |
| `dialog` | `components/ui/dialog.gox` | Accessible modal dialog with backdrop overlay and close action |
| `badge` | `components/ui/badge.gox` | Status indicator badges (`default`, `success`, `warning`, `destructive`) |
| `dropdown` | `components/ui/dropdown.gox` | Dropdown action menu with items and divider elements |
| `table` | `components/ui/table.gox` | Responsive data table with styled header, alternating rows, and hover highlights |

---

## 🔄 Progressive Server Actions (`pkg/action`)

Server Actions allow you to run backend Go functions directly from HTML forms with seamless progressive enhancement:

```go
package actions

import (
	"fmt"
	"github.com/misbakhul29/goks/pkg/action"
	"myapp/models"
)

func init() {
	// Register a named Server Action
	action.Register("createSubscriber", func(ctx *action.Context) (any, error) {
		email := ctx.FormData.Get("email")
		if email == "" {
			return nil, fmt.Errorf("email is required")
		}

		// Direct database access on the server
		// orm.Create(orm.DB, &models.Subscriber{Email: email})

		return map[string]string{"message": "Subscribed successfully!"}, nil
	})
}
```

In your `.gox` view:
```html
<form action={action.URL("createSubscriber")} method="POST" class="space-y-4">
	<input type="email" name="email" placeholder="name@example.com" required class="..." />
	<button type="submit" class="...">Subscribe</button>
</form>
```

- **In the Browser (WASM)**: Automatically intercepts `<form>` submits to `/__goks_action`, performs background `fetch()`, and updates the UI without full page refreshes.
- **Zero-JS Fallback**: If JavaScript/WASM is unavailable, submits as a standard HTML form POST and redirects back with HTTP 303.
- **CSRF Protection**: Automatically validates request origins against server host headers.

---

## 🏝️ Islands Architecture & Selective Hydration

GoKS implements selective hydration to keep web pages fast and lightweight:

1. **Pure Static Pages (0 KB WASM)**: If a page contains no event handlers (`onClick`, etc.) and no client islands, GoKS outputs **100% pure HTML**. Neither `wasm_exec.js` nor `app.wasm` is downloaded.
2. **Interactive Islands**: When interactive elements are detected or components embed `component.ClientBase`, GoKS hydrates the WebAssembly runtime for that page.

```go
type InteractiveWidget struct {
	component.ComponentBase
	component.ClientBase // Enables WASM hydration for this island
}
```

---

## 🗄️ Built-in ORM with SQLite

GoKS includes a database ORM with zero external configuration required:

```go
package main

import (
	"log"
	"github.com/misbakhul29/goks/pkg/orm"
)

type Post struct {
	orm.Model
	Title   string `db:"title"`
	Content string `db:"content"`
	Author  string `db:"author"`
}

func main() {
	// 1. Zero-config SQLite database (creates app.db automatically)
	db, err := orm.OpenSQLite("app.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 2. Insert record
	newPost := &Post{Title: "Hello GoKS", Content: "Building fullstack apps in Go", Author: "Alex"}
	_ = orm.Create(db, newPost)

	// 3. Query records with fluent API
	posts, err := orm.Query[Post]().
		Where("author", "=", "Alex").
		OrderBy("created_at DESC").
		Limit(10).
		Find(db)
}
```

Supports **SQLite**, **PostgreSQL**, and **MySQL**.

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

## 🚢 Deployment Guide

GoKS's `--standalone` build embeds all assets (WASM binary, compiled CSS, and public static files) directly into a **single executable binary**.

```bash
goks build --standalone
# Output generated at: .goks/standalone/server
```

### 1. VPS / Cloud VM (Ubuntu, Debian, etc.)

```bash
# On your local machine:
goks build --standalone
scp .goks/standalone/server user@your-server-ip:/opt/myapp/server

# On your server:
chmod +x /opt/myapp/server
PORT=80 /opt/myapp/server
```

### 2. Docker (Ultra-Minimal Scratch Image, ~15MB)

```dockerfile
# Build stage
FROM golang:1.22-alpine AS builder
WORKDIR /app
RUN apk add --no-cache curl
RUN go install github.com/misbakhul29/goks@latest
COPY . .
RUN goks build --standalone

# Production scratch stage
FROM scratch
COPY --from=builder /app/.goks/standalone/server /server
EXPOSE 3000
ENTRYPOINT ["/server"]
```

### 3. Deploying to Vercel

Because Vercel serverless build images do not come with a Go compiler by default, deploy GoKS to Vercel via **GitHub Actions** using the Vercel CLI:

Create `.github/workflows/deploy.yml`:

```yaml
name: Deploy to Vercel
on:
  push:
    branches: [main, master]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Install GoKS
        run: go install github.com/misbakhul29/goks@latest

      - name: Build GoKS Standalone
        run: goks build --standalone

      - name: Deploy to Vercel
        uses: amondnet/vercel-action@v25
        with:
          vercel-token: ${{ secrets.VERCEL_TOKEN }}
          vercel-org-id: ${{ secrets.VERCEL_ORG_ID }}
          vercel-project-id: ${{ secrets.VERCEL_PROJECT_ID }}
          working-directory: .goks/standalone
```

### 4. Railway, Fly.io & Render

For container platforms, simply point them to the Dockerfile above or run the standalone binary directly:

**Fly.io:**
```bash
fly launch --dockerfile Dockerfile
fly deploy
```

---

## 🤝 Contributing

Contributions, issues, and feature requests are welcome!
Feel free to open an issue or submit a pull request on [GitHub](https://github.com/misbakhul29/goks).

---

## 📄 License

This project is licensed under the [MIT License](LICENSE).
