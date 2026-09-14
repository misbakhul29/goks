# GoKS Architecture (Overview)

This is the living high-level architecture reference. Package-specific detail
lives under `.agents/architecture/`. Fundamental decisions live under
`docs/adr/`. This file must stay in sync with the actual repository layout —
update it in the same change that restructures code.

## Layered View

```
                     GoKS
                       |
        +--------------+--------------+
        |                             |
     Runtime                      Compiler
   (server + client)            (GOX -> Go, Tailwind)
        |                             |
  +-----+-----+-----+           +-----+-----+
  |     |     |     |           |     |     |
 HTTP  ORM  Auth  Realtime     GOX   WASM   CSS
 (router) (orm) (auth/rbac)   (compiler) (component/runtime/client) (tailwind)
        |
   Developer API (pkg/*)
        |
   GoKS Application (user code: app/, components/)
        |
   Single Standalone Binary (via `goks build --standalone`)
```

## Current Package Responsibilities

| Package | Responsibility |
| --- | --- |
| `pkg/router` | File-system routing (`app/page.gox`, `app/layout.gox`) + programmatic routes. |
| `pkg/action` | Server actions: dual-mode (WASM `fetch()` + no-JS HTML form fallback), CSRF. |
| `pkg/component` | Component model, hooks (`UseState`, ...), reconciliation, SSR, islands, Suspense, and streaming. |
| `internal/compiler` | GOX syntax -> Go code generation in an isolated workspace. |
| `internal/generator` | File-system route and entrypoint generation. |
| `pkg/orm` | Query builder, migrations, soft delete, timestamps, SQLite support. |
| `pkg/auth`, `pkg/auth/oauth` | JWT/session authentication primitives and OAuth2 providers. |
| `pkg/rbac` | Role-based access control middleware. |
| `pkg/ws` | WebSocket hub, typed event routing, rooms. |
| `pkg/store` | Reactive global state store for client WASM. |
| `pkg/rpc` | RPC plumbing bridging server actions and the WASM client. |
| `pkg/cache`, `pkg/env`, `pkg/font`, `pkg/html`, `pkg/image`, `pkg/metadata` | Supporting infrastructure. |
| `pkg/studio` | Embedded development dashboard for routes, actions, RPC, database, migrations, and runtime telemetry. |
| `internal/cli` | `goks` CLI commands: `new`, `dev`, `build`, `start`, `ui`, `generate`, `page`, `db`, `studio`, `export`, and `lsp`. |
| `internal/version` | Single source of the framework version used by tooling and runtime diagnostics. |
| `internal/watcher`, `internal/livereload` | Dev server hot reload. |
| `internal/lsp` | Editor/LSP support for `.gox` files. |
| `runtime/server`, `runtime/client` | Glue that wires a compiled GoKS app together on server and in the WASM client. |

## Request Lifecycle (server-rendered page)

```
HTTP request
  -> middleware stack (Logger, CORS, Secure, RequestID, Compress, MaxBytes, Timeout)
  -> pkg/router match (file-system route)
  -> pkg/auth / pkg/rbac (if protected)
  -> compiled page component render (pkg/component, generated from .gox)
  -> pkg/html render to response
  -> (if page has interactive components) inline hydration bootstrap for WASM island
```

## Server Action Lifecycle

```
Client event
  -> WASM present?  -> pkg/rpc call -> pkg/action handler -> CSRF check -> business logic
  -> WASM absent?    -> HTML form POST -> pkg/action handler -> CSRF check -> business logic
  -> response -> update DOM (WASM) or full page reload (no-JS fallback)
```

## Build Lifecycle

```
goks dev:    internal/watcher -> internal/compiler (GOX->Go) -> go build (server)
                                                              -> tinygo/go build (WASM)
                                                              -> Tailwind standalone compile
                                                              -> internal/livereload push

goks build --standalone: same pipeline, output = single binary embedding
  WASM + CSS + HTML assets.
```

## Keep This File Honest

If you add, remove, split, or rename a package, or change a lifecycle above,
update this file (and the relevant `.agents/architecture/*.md`) in the same
change. A stale architecture doc is worse than none, because agents will
trust it.
