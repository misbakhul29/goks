# GoKS (Go Kickstart)

**GoKS is a modern, open-source fullstack web framework built entirely in Go.** 
It enables developers to build fast, type-safe web applications using Go on both 
backend and frontend (WebAssembly), eliminating the need for Node.js, npm, or 
JavaScript. GoKS combines the ergonomics of React/Next.js with the performance 
and single-binary deployment of Go, making it ideal for building microservices, 
APIs, and progressive web applications.

**Key Technologies:** Go 1.22+, WebAssembly (WASM), TinyGo, Tailwind CSS v4, 
SQLite, PostgreSQL, MySQL, Server Actions.

<div align="center">

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://golang.org)
[![Release](https://img.shields.io/badge/release-v0.15.0-6366F1?style=for-the-badge&logo=github)](https://github.com/misbakhul29/goks/releases)
[![License](https://img.shields.io/badge/license-MIT-10B981?style=for-the-badge)](LICENSE)
[![WASM](https://img.shields.io/badge/WebAssembly-Enabled-654FF0?style=for-the-badge&logo=webassembly&logoColor=white)](https://webassembly.org)
[![Tailwind](https://img.shields.io/badge/Tailwind_CSS-v4-06B6D4?style=for-the-badge&logo=tailwindcss&logoColor=white)](https://tailwindcss.com)

**The Modern Fullstack Go Web Framework.**  
*Backend APIs + WebAssembly Frontend, 100% Type-Safe Go. Zero Node.js or JavaScript Required.*

[Getting Started](#-quick-start) • [Features](#-features) • [Why GoKS?](#-why-goks) • [UI Components](#-goks-ui-component-system) • [OAuth2 Login](#-oauth2-social-login-providers-pkgauthoauth) • [Streaming SSR](#-streaming-ssr--suspense-componentsuspense) • [DevTools](#-goks-studio--embedded-devtools-goks-studio) • [Image Optimizer](#-image-optimizer--component-uiimage) • [API Routes](#-file-based-api-route-handlers) • [Database Migrations](#-database-migrations-cli-goks-db) • [Deployment](#-deployment-guide)

</div>

---

## ⚡ Why GoKS?

GoKS bridges the gap between modern React/Next.js developer ergonomics and the legendary performance, type-safety, and single-binary deployment of Go.

| Feature | GoKS | Next.js / Remix | templ + HTMX | Fiber / Gin |
| :--- | :---: | :---: | :---: | :---: |
| **Language Stack** | **100% Go** (Fullstack) | TypeScript / JS | Go + HTML Attributes | Go (Backend Only) |
| **Frontend Runtime** | **WebAssembly (Go)** | JS Virtual DOM | Browser Native / DOM swap | None / Raw Templates |
| **Node.js / npm Required?** | **❌ No (Zero npm)** | ✅ Required | ❌ No | ❌ No |
| **OAuth2 Social Login** | **✅ Built-in (Google, GitHub, PKCE)** | ⚠️ NextAuth / Clerk | ❌ None | ❌ Manual integration |
| **Streaming SSR & Suspense** | **✅ Built-in (`component.Suspense`)** | ✅ React 18 Suspense | ❌ None | ❌ None |
| **File-Based API Routes** | **✅ Built-in (`app/api/**/route.go`)** | ✅ Yes | ❌ No | ❌ Manual |
| **Image Optimizer** | **✅ Built-in (`<ui.Image />` / `pkg/image`)** | ✅ `next/image` | ❌ None | ❌ None |
| **Database Migrations** | **✅ Built-in (`goks db`)** | ⚠️ Prisma / Drizzle | ⚠️ Goose / Migrate | ❌ None |
| **Embedded DevTools** | **✅ Built-in (`/__goks` / `goks studio`)** | ⚠️ Community extension | ❌ None | ❌ None |
| **Server Actions** | **✅ Built-in (`pkg/action`)** | ✅ Yes | ⚠️ Partial (HTMX triggers) | ❌ Manual endpoints |
| **Selective Hydration** | **✅ Islands (0-WASM static)** | ⚠️ Partial (React RSC) | ❌ N/A | ❌ N/A |
| **Component Generator** | **✅ `goks ui` (shadcn-style)** | ✅ `shadcn/ui` | ❌ None | ❌ None |
| **Tailwind CSS v4** | **✅ Built-in standalone** | ⚠️ Needs Node.js/npm | ⚠️ External CLI setup | ⚠️ External setup |
| **Production Artifact** | **Single Standalone Binary** | Node runtime + node_modules | Single Binary | Single Binary |

---

## ✨ Features

- **🚀 GOX Syntax (`.gox`):** Declarative, JSX-like markup directly inside Go (`<div><h1>{title}</h1></div>`, `<ui.Button />`). Compiled directly into Go code in an isolated workspace.
- **🔐 OAuth2 Social Login Providers (`pkg/auth/oauth`):** Zero-dependency social authentication supporting Google and GitHub. Features PKCE (RFC 7636 S256 code challenge) protection, cryptographic random state verification (timing-attack resistant via `subtle.ConstantTimeCompare`), automatic token exchange, and unified user profile mapping.
- **🌊 Streaming SSR & Suspense (`component.Suspense`):** Out-of-order streaming server-side rendering over chunked HTTP transfer encoding. Send the fast HTML shell immediately while slow asynchronous database queries or external API calls resolve in parallel in Go goroutines, progressively replacing loading skeleton fallbacks via native zero-dependency `<template>` DOM swaps.
- **🛠️ GoKS Studio / Embedded DevTools (`/__goks` / `goks studio`):** Built-in development dashboard to inspect discovered routes (Pages & REST APIs), browse database tables and schemas, monitor registered Server Actions & RPC methods, view migration statuses, and track live runtime memory/goroutine metrics. Automatically disabled in production with zero data leakage.
- **🖼️ Image Optimizer & Component (`<ui.Image />` / `pkg/image`):** Automatic on-demand image resizing, high-performance pure Go bilinear interpolation, responsive `srcset` generation, `304 Not Modified` ETag caching, and layout shift (CLS) prevention.
- **🗄️ Database Migrations CLI (`goks db`):** Full database lifecycle management (`make:migration`, `migrate`, `rollback`, `status`, `seed`) supporting SQLite, PostgreSQL, and MySQL with transactional atomicity and batch rollbacks.
- **🌐 File-Based API Route Handlers (`app/api/**/route.go`):** Next.js App Router style backend API endpoints. Simply export standard HTTP method functions (`GET`, `POST`, `PUT`, `DELETE`, `PATCH`, `OPTIONS`, `HEAD`) with full path parameters (`:id` or `[id]`), query params, JSON body binding, and automatic recovery. API routes are strictly server-side and never bundled into client WASM.
- **📦 Static Site Generation (SSG, `goks export`):** Pre-render your entire application to static HTML and deploy to Cloudflare Pages, GitHub Pages, Vercel, or AWS S3 with zero server running costs.
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
  - `goks export` — Pre-render full static HTML website for CDN/serverless hosting.
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
| `image` | `components/ui/image.gox` | Responsive, optimized image with layout shift protection (CLS) and on-demand resizing |

---

## 🌊 Streaming SSR & Suspense (`component.Suspense`)

GoKS provides built-in out-of-order **Streaming Server-Side Rendering (Streaming SSR)** inspired by React 18 Suspense, powered natively by Go goroutines, `Transfer-Encoding: chunked`, and zero-dependency `<template>` DOM swaps.

Instead of blocking the entire page response until slow database queries or upstream microservices finish, GoKS immediately streams the HTML shell with fallback skeletons. As goroutines resolve their asynchronous tasks, replacement chunks are flushed over the persistent HTTP connection and swapped into the DOM instantly.

### Usage Example:
```go
package page

import (
	"context"
	"time"

	"github.com/misbakhul29/goks/pkg/component"
)

func ProfilePage() *component.Node {
	return component.H("div", component.Props{"class": "max-w-xl mx-auto p-6"},
		component.H("h1", component.Props{"class": "text-2xl font-bold mb-4"}, 
			component.Text("User Account"),
		),

		// Suspense boundary with asynchronous resolver
		component.Suspense(component.SuspenseProps{
			Fallback: component.H("div", component.Props{"class": "animate-pulse space-y-3"},
				component.H("div", component.Props{"class": "h-4 bg-zinc-800 rounded w-3/4"}),
				component.H("div", component.Props{"class": "h-4 bg-zinc-800 rounded w-1/2"}),
			),
			Async: func(ctx context.Context) (*component.Node, error) {
				// Simulating slow database query or external API
				time.Sleep(150 * time.Millisecond)
				
				return component.H("div", component.Props{"class": "p-4 bg-zinc-900 border border-zinc-800 rounded-lg"},
					component.H("h2", component.Props{"class": "text-lg font-semibold text-indigo-400"}, 
						component.Text("Alex Mercer"),
					),
					component.H("p", component.Props{"class": "text-sm text-zinc-400"}, 
						component.Text("alex@goks.dev • Pro Tier"),
					),
				), nil
			},
		}),
	)
}
```

### Streaming Route Handler (`c.StreamHTML`):
```go
func GET(c *router.Context) error {
	return c.StreamHTML(func(w io.Writer, flusher http.Flusher) error {
		_, err := component.RenderToStream(c.Request().Context(), w, flusher, ProfilePage())
		return err
	})
}
```

### How It Works Under The Hood:
1. **Instant TTFB**: Initial HTML shell arrives in single-digit milliseconds with `<div id="goks-s-1" data-goks-suspense="pending">...skeleton...</div>`.
2. **Concurrent Goroutines**: Each `Async` resolver runs concurrently in isolated Go goroutines respecting `context.Context` request lifecycles.
3. **Chunked Streaming**: When a task completes, a `<template id="goks-c-1">` chunk containing the resolved HTML and a lightweight 2-line inline DOM swap script is flushed.
4. **Zero Client Runtime**: Requires no heavy client-side JavaScript framework. Native browser DOM APIs (`replaceWith`, `cloneNode`) execute the swap seamlessly.
5. **Progressive Fallback**: If JavaScript is disabled in the browser, the accessible skeleton/fallback remains rendered without broken page states.

---

## 🖼️ Image Optimizer & Component (`<ui.Image />`)

GoKS includes built-in, on-demand image optimization and a Next.js-style `<ui.Image />` component. Images are automatically resized via pure Go bilinear interpolation, cached on disk with HTTP `304 Not Modified` ETags, and served with responsive `srcset` definitions:

```html
<!-- Inside any .gox component -->
<ui.Image
    src="/photos/banner.jpg"
    alt="Hero Banner"
    width={1200}
    height={600}
    priority={true}
    class="rounded-2xl shadow-xl w-full"
/>
```

Or using the standard package directly:
```go
import "github.com/misbakhul29/goks/pkg/image"

// In a component render:
image.New(image.Props{
    Src:      "/photos/banner.jpg",
    Alt:      "Hero Banner",
    Width:    1200,
    Height:   600,
    Quality:  80,
    Priority: true,
}).Render()
```

### Key Capabilities & Security:
- **Zero CLS (Cumulative Layout Shift)**: Automatically computes and sets CSS `aspect-ratio` to reserve screen space while images load.
- **Responsive `srcset`**: Generates device-tailored breakpoints (`640w`, `750w`, `828w`, `1080w`, `1200w`, `1920w`).
- **High-Performance Pure Go**: Bilinear scaling without external C libraries (`CGO_ENABLED=0` friendly).
- **Hardened Security**:
  - **Path Traversal Protection (CWE-22)**: Confines file access strictly inside `public/`.
  - **SSRF Prevention (CWE-918)**: Rejects private IP ranges (`127.0.0.1`, `10.0.0.0/8`, `169.254.169.254`, etc.) and requires domain whitelisting for remote images.
  - **Decompression Bomb Protection (CWE-409)**: Inspects image dimensions before full allocation, rejecting dimensions exceeding 4096px or files larger than 20MB.
  - **Cache Flooding Mitigation**: Snaps arbitrarily requested widths to standard responsive breakpoints to prevent DoS attacks.

---

## 🌐 File-Based API Route Handlers

GoKS brings Next.js App Router-style file-based routing to backend APIs. Simply place a `route.go` file inside any folder in `app/` (typically `app/api/**/route.go`) and export standard HTTP method functions:

```go
// app/api/users/route.go
package users

import (
	"github.com/misbakhul29/goks/pkg/router"
)

type CreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// GET /api/users
func GET(c *router.Context) error {
	return c.JSON([]map[string]string{
		{"id": "1", "name": "Alice"},
		{"id": "2", "name": "Bob"},
	})
}

// POST /api/users
func POST(c *router.Context) error {
	var body CreateUserRequest
	if err := c.Bind(&body); err != nil {
		return c.Status(400).JSON(map[string]string{"error": "Invalid request body"})
	}
	return c.Status(201).JSON(map[string]any{
		"status": "created",
		"user":   body,
	})
}
```

### Dynamic & Wildcard URL Parameters
Create dynamic endpoints using Next.js-style `[id]` or GoKS-native `_id` folder names:

```go
// app/api/users/[id]/route.go (or app/api/users/_id/route.go)
package id

import "github.com/misbakhul29/goks/pkg/router"

// GET /api/users/:id
func GET(c *router.Context) error {
	userId := c.Param("id")
	return c.JSON(map[string]string{"user_id": userId})
}

// DELETE /api/users/:id
func DELETE(c *router.Context) error {
	userId := c.Param("id")
	// orm.Delete(...)
	return c.Status(204).Text("")
}
```

### Wildcard Catch-All Routes
Catch-all routes using `[...slug]` capture full remaining URL paths:

```go
// app/api/files/[...path]/route.go
package path

import "github.com/misbakhul29/goks/pkg/router"

// GET /api/files/*path (e.g., /api/files/docs/2026/report.pdf)
func GET(c *router.Context) error {
	filePath := c.Param("path") // "docs/2026/report.pdf"
	return c.JSON(map[string]string{"requested_file": filePath})
}
```

- **Supported HTTP Methods**: `GET`, `POST`, `PUT`, `DELETE`, `PATCH`, `HEAD`, `OPTIONS`.
- **Zero WASM Leakage**: API route handlers are compiled strictly into the Go backend server binary and are **never bundled into client WebAssembly** or downloaded by the browser.
- **Built-in Resilience**: Protected by `router.Recover()` to ensure handler panics return HTTP 500 without crashing your server.

---

## 🔐 OAuth2 Social Login Providers (`pkg/auth/oauth`)

GoKS provides built-in, zero-dependency OAuth2 social authentication for **Google**, **GitHub**, and custom providers. It features **PKCE (RFC 7636)** protection, cryptographic random state verification with constant-time equality checks against timing attacks, and unified user profile mapping.

### 1. Initiate Social Login Flow
```go
// app/api/auth/login/[provider]/route.go
package provider

import (
	"github.com/misbakhul29/goks/pkg/auth/oauth"
	"github.com/misbakhul29/goks/pkg/router"
)

var googleAuth = oauth.Google(oauth.Config{
	ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
	ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
	RedirectURL:  "http://localhost:3000/api/auth/callback/google",
})

// GET /api/auth/login/google
func GET(c *router.Context) error {
	state, _ := oauth.GenerateState()
	verifier, challenge, _ := oauth.GeneratePKCE()

	// Store state and verifier in secure, HTTP-only session cookies
	c.SetHeader("Set-Cookie", "oauth_state="+state+"; Path=/; HttpOnly; SameSite=Lax")

	authURL := googleAuth.AuthURL(oauth.AuthOptions{
		State:         state,
		CodeChallenge: challenge,
		Prompt:        "select_account",
	})
	return c.Redirect(authURL)
}
```

### 2. Handle OAuth2 Callback & Profile Mapping
```go
// app/api/auth/callback/[provider]/route.go
package callback

import (
	"github.com/misbakhul29/goks/pkg/auth/oauth"
	"github.com/misbakhul29/goks/pkg/router"
)

// GET /api/auth/callback/google
func GET(c *router.Context) error {
	expectedState := getCookie(c.Request(), "oauth_state")
	receivedState := c.Query("state")

	// Verify state token using constant-time comparison (prevents CSRF & timing attacks)
	if !oauth.VerifyState(expectedState, receivedState) {
		return c.Status(400).Text("Invalid OAuth state parameter")
	}

	code := c.Query("code")
	token, err := googleAuth.Exchange(c.Request().Context(), code)
	if err != nil {
		return c.Status(400).Text("Failed to exchange token: " + err.Error())
	}

	// Fetch unified user profile
	user, err := googleAuth.UserInfo(c.Request().Context(), token)
	if err != nil {
		return c.Status(500).Text("Failed to fetch user profile")
	}

	// user.ID, user.Email, user.Name, user.AvatarURL, user.Provider
	return c.JSON(user)
}
```

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

## 🗄️ Database Migrations CLI (`goks db`)

GoKS includes a full-featured, zero-dependency database migrations and seeding engine supporting **SQLite**, **PostgreSQL**, and **MySQL** with transactional atomicity and batch rollback tracking.

### 1. Create a Migration
Generate a timestamped SQL migration file inside `migrations/`:
```bash
goks db make:migration create_users_table
# Creates: migrations/20260914120000_create_users_table.sql
```

Migration files contain forward (`Up`) and reverse (`Down`) SQL statements separated by `-- +goks Up` and `-- +goks Down`:
```sql
-- +goks Up
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- +goks Down
DROP TABLE IF EXISTS users;
```

### 2. Run Pending Migrations
Applies all pending migrations inside a single database transaction per file:
```bash
goks db migrate
```
This automatically initializes the `goks_migrations` tracking table and records each migration's timestamp and batch ID.

### 3. Check Migration Status
View migration history, batch IDs, and applied timestamps:
```bash
goks db status
```
```text
  Migration                                Batch  Status
  ----------------------------------------------------------------------
  20260914120000_create_users_table.sql    1      Applied (2026-09-14 12:01:00)
  20260914123000_create_posts_table.sql    1      Applied (2026-09-14 12:01:00)
  20260914130000_add_avatar_to_users.sql   -      Pending
```

### 4. Rollback Migrations
Revert the last batch of migrations (or specify `--steps=N`):
```bash
# Rollback last migration batch:
goks db rollback

# Rollback exactly 2 migrations:
goks db rollback --steps=2
```

### 5. Seed Database
Execute raw SQL seed files from `seeds/*.sql` in alphabetical order:
```bash
goks db seed
```

---

## 🛠️ GoKS Studio / Embedded DevTools (`/__goks`)

GoKS includes a built-in interactive developer dashboard to inspect your fullstack application in real-time.

```bash
# Launch GoKS Studio directly in your browser:
goks studio

# Or access it automatically while running the dev server:
goks dev
# Open: http://localhost:3000/__goks
```

### Features:
1. **📊 Real-time Runtime Telemetry**: Live memory allocation, active goroutines, GC cycles, server uptime, and Go runtime environment.
2. **🗺️ Routes Explorer**: Visual catalog of all Page routes (`app/**/page.gox`) and backend API routes (`app/api/**/route.go`) with HTTP methods (`GET`, `POST`, `PUT`, `DELETE`, `PATCH`).
3. **🗄️ Database Studio**: Live SQLite/PostgreSQL/MySQL table browser. Inspect schema, column types, total row counts, and browse records with paginated data views.
4. **⚡ Server Actions & RPC Catalog**: List of registered Server Actions (`pkg/action`) and RPC methods (`pkg/rpc`) with clickable endpoint paths.
5. **📜 Migrations Monitor**: Track migration files, applied timestamps, and batch numbers.
6. **🔒 Production-Safe**: GoKS Studio is automatically restricted to development mode (`DevMode: true`). In production builds, all studio endpoints immediately return 404 with zero data leakage.

---

## ⌨️ CLI Reference

| Command | Description |
| :--- | :--- |
| `goks new <app>` | Create a new GoKS application |
| `goks dev [-p port] [--compiler=tinygo]` | Start development server with hot-reload (default: port 3000) |
| `goks studio [-p port] [--no-open]` | Launch the embedded GoKS Studio DevTools dashboard |
| `goks page <route>` | Scaffold a new `.gox` page in `app/<route>/page.gox` |
| `goks generate page <route>` | Alias to generate a page |
| `goks generate component <Name>` | Scaffold a reusable component in `app/components/<name>.gox` |
| `goks generate model <Name>` | Scaffold an ORM model in `models/<name>.go` |
| `goks ui list` | List all available UI components |
| `goks ui add <name... \| all>` | Add styled Tailwind components to `components/ui/` |
| `goks db make:migration <name>` | Scaffold a new timestamped SQL migration file |
| `goks db migrate` | Run all pending migrations in atomic transactions |
| `goks db rollback [--steps=N]` | Roll back the last batch or specified number of migrations |
| `goks db status` | Display status of all migrations |
| `goks db seed` | Run all SQL seeders from `seeds/` |
| `goks export [-o out] [--serve]` | Export as 100% static HTML site (SSG) for Cloudflare/GitHub Pages |
| `goks build [--compiler=tinygo]` | Compile WASM bundle, Tailwind CSS, and production server binary |
| `goks build --standalone` | Build a **single self-contained binary** with all assets embedded |
| `goks start [port]` | Run production server (built inside `.goks/build/`) |
| `goks version` | Display current GoKS version |

---

## 🚢 Deployment Guide

### 1. Static Site Hosting (`goks export`) ✨
Pre-render every route into pure HTML and bundle assets into `out/`:
```bash
# Export static website to out/
goks export

# Preview the static site locally before deploying:
goks export --serve -p 3000
```
Deploy the generated `out/` folder directly to **Cloudflare Pages**, **GitHub Pages**, **Netlify**, or **AWS S3 / CloudFront** for free static hosting with zero server running costs.

### 2. Standalone Binary (`goks build --standalone`)
Produces a **single binary** at `.goks/standalone/server` with all assets embedded:
```bash
goks build --standalone
```

#### A. VPS / Cloud VM (Ubuntu, Debian, etc.)

```bash
# On your local machine:
goks build --standalone
scp .goks/standalone/server user@your-server-ip:/opt/myapp/server

# On your server:
chmod +x /opt/myapp/server
PORT=80 /opt/myapp/server
```

#### B. Docker (Ultra-Minimal Scratch Image, ~15MB)

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

#### C. Deploying to Vercel

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

#### D. Railway, Fly.io & Render

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
