# GoKS CLI Reference

The `goks` binary provides unified developer tooling for project scaffolding, local development, asset compilation, database migrations, and production releases.

## Exit Code Contract

All `goks` commands follow standard UNIX exit code conventions suitable for CI/CD scripting:
- `0`: Success
- `1`: User error (invalid flag, missing files, or bad configuration)
- Non-zero: Build, compilation, or migration failure

Diagnostics and actionable errors are written to standard error (`stderr`).

---

## Commands

### `goks new <app-name>`
Scaffolds a new, production-ready GoKS application.

```bash
goks new my-app
goks new my-app --module=github.com/myorg/my-app
```

**Flags:**
- `-m, --module <string>`: Go module path (default: `github.com/user/<app-name>`).

**Validation:**
- Rejects empty names and names containing path separators or directory traversal characters (`..`, `/`, `\`).
- Fails safely with a clear error if the target directory already exists (overwrite protection).

---

### `goks dev`
Starts the local development server with automatic file watching, incremental GOX transpilation, live CSS compilation with Tailwind CSS v4, and browser live reload.

```bash
goks dev
goks dev --port 8080
```

**Flags:**
- `-p, --port <int>`: HTTP port to listen on (default: `3000`).
- `--compiler <string>`: WASM compiler (`go` [default] or `tinygo`).

---

### `goks build`
Compiles the application for production deployment. Transpiles `.gox` files, compiles the WebAssembly bundle, processes CSS, and outputs the server binary.

```bash
# Standard production build
goks build

# Standalone self-contained binary (like Next.js output: standalone)
goks build --standalone

# Ultra-compact build using TinyGo
goks build --compiler=tinygo
```

**Flags:**
- `--standalone`: Bundles WASM, CSS, JS runtime, and `public/` assets directly into the server binary using Go's `//go:embed`.
- `--compiler <string>`: WASM compiler (`go` [default] or `tinygo`).

---

### `goks start`
Starts the compiled production server.

```bash
goks start
PORT=8080 goks start
```

---

### `goks export`
Performs static site generation (SSG), exporting all static routes to pure HTML, CSS, and WASM assets in `dist/`.

```bash
goks export
goks export --dir ./my-app --out ./static-site
```

**Flags:**
- `--dir <string>`: Project root directory (default: `.`).
- `--out <string>`: Output directory (default: `dist`).
- `--compiler <string>`: WASM compiler (`go` [default] or `tinygo`).

---

### `goks db`
Manages database migrations.

```bash
goks db create <migration-name>
goks db up
goks db down
goks db status
```

**Features:**
- Automatic rollback on error during atomic transactions.
- Path traversal protection on migration files.

---

### `goks page <route-path>`
Generates a new page and layout boilerplate inside `app/`.

```bash
goks page dashboard
goks page users/[id]
```

---

### `goks ui <component-name>`
Scaffolds a reusable UI component.

```bash
goks ui button
goks ui card
```

---

### `goks studio`
Starts the GoKS Studio development dashboard for inspecting routes, ORM models, telemetry, and live state.

```bash
goks studio --port 4000
```

---

### `goks lsp`
Runs the Language Server Protocol (LSP) daemon over standard I/O for IDE integrations (VS Code, Neovim) providing auto-completion, hover documentation, syntax diagnostics, and code formatting for `.gox` files.

```bash
goks lsp
```
