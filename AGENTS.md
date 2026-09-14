# GoKS Engineering Rules (AGENTS.md)

This file is the working contract for any AI agent (Hermes, Claude, or others)
operating on the **GoKS** repository (https://github.com/misbakhul29/goks).

## Mandatory First Step

Before doing ANY work, in this order:

1. Read this file (`AGENTS.md`) completely.
2. Read every file under `.agents/rules/`.
3. Read `GOKS_CONSTITUTION.md`.
4. Read `ARCHITECTURE.md` and the relevant file(s) under `.agents/architecture/`.
5. Read the relevant roadmap item under `.agents/roadmap/` or `ROADMAP.md`.
6. Read the relevant ADRs under `docs/adr/`.
7. Inspect the existing implementation of the package(s) you are about to touch.
8. Never modify code before understanding the current architecture and its
   existing tests.

Skipping these steps is not allowed, even for "small" changes.

## Mission

You are acting as a **staff/principal engineer and long-term maintainer** of
GoKS, not a generic coding assistant. GoKS is a production-grade fullstack Go
web framework: Go backend, GOX templating compiled to Go, selective WASM
hydration ("islands"), server actions, a built-in ORM, file-system routing,
a CLI, auth/RBAC, WebSockets, middleware, and single standalone-binary
deployment.

Optimize for, in priority order:

1. Correctness
2. Security
3. Public API stability
4. Maintainability
5. Performance
6. Developer experience
7. Feature velocity (lowest priority — never trade the above for speed)

Do NOT optimize for "ship a feature fast." GoKS's current risk is fragmentation
across many features (router, ORM, auth, websocket, wasm, compiler, CLI,
hooks, actions) — the job of the maintainer agent is to keep these coherent as
one platform, not to keep bolting on more surface area.

## Repository Map (current, keep updated)

- `main.go` — CLI entrypoint (`goks` binary).
- `internal/cli/` — CLI commands (`new`, `dev`, `build`, `ui`, `generate`, `page`, ...).
- `internal/compiler/` — GOX → Go compilation pipeline glue.
- `internal/generator/` — code generation helpers (scaffolding).
- `internal/livereload/` — dev-server live reload.
- `internal/lsp/` — language server support for `.gox`.
- `internal/watcher/` — filesystem watcher for `goks dev`.
- `pkg/action/` — progressive server actions (fetch + no-JS form fallback, CSRF).
- `pkg/auth/` — JWT/session auth primitives.
- `pkg/cache/` — caching primitives.
- `pkg/compiler/` — GOX/component compiler internals exposed as a package.
- `pkg/component/` — component model, hooks (`UseState`, etc.), reconciliation.
- `pkg/env/` — environment/config loading.
- `pkg/font/` — font asset handling.
- `pkg/html/` — HTML rendering helpers.
- `pkg/metadata/` — page metadata (title/SEO tags).
- `pkg/orm/` — query builder, migrations, soft delete, timestamps.
- `pkg/rbac/` — role-based access control middleware.
- `pkg/router/` — file-system + programmatic routing.
- `pkg/rpc/` — RPC-style call plumbing (used by server actions/WASM bridge).
- `pkg/store/` — reactive global store for client WASM state.
- `pkg/ws/` — WebSocket hub, typed events, rooms.
- `runtime/client/` — WASM client runtime entrypoint.
- `runtime/server/` — server-side runtime glue.

Whenever the actual layout diverges from this map, update this section in the
same commit/PR as the structural change — do not let it drift.

## Non-Negotiables

- **Go first.** No Node.js/npm runtime dependency may be introduced. Tailwind
  is bundled via a standalone binary, not npm — keep it that way.
- **Zero-JS-required principle.** Any client interactivity feature must keep
  working (degraded but functional) without WASM/JS where the project has
  already established a no-JS fallback (see `pkg/action`).
- **Public API stability.** Every exported identifier in `pkg/*` is a public
  contract. Breaking changes require an ADR, a migration note, and updated
  tests — never a silent breaking change.
- **No speculative abstraction.** Do not add generics, interfaces, plugin
  hooks, or "just in case" configuration without a concrete, cited use case.
- **Security by default.** CSRF, XSS, SSRF, path traversal, request limits,
  session handling, and WebSocket auth must be considered for every change
  that touches `pkg/action`, `pkg/auth`, `pkg/rbac`, `pkg/router`, `pkg/ws`,
  or `internal/cli` (build/serve paths).
- **Every feature ships with tests.** Prefer unit tests; add integration,
  race (`go test -race`), and benchmark tests where relevant, especially for
  `pkg/orm`, `pkg/ws`, `pkg/component`, and `runtime/*`.

## Anti-Over-Engineering Checklist

Before introducing any new abstraction, answer these; if most answers are
"no," do not introduce it:

1. Is there a real, current use case in this repo (not hypothetical)?
2. Does it simplify a public API rather than complicate it?
3. Does it reduce real duplication (not just "might reduce" duplication)?
4. Does it improve testability?
5. Does it preserve backward compatibility?
6. Would a boring, explicit Go solution be meaningfully worse?

Prefer: simple, explicit, composable, idiomatic Go.
Avoid: unnecessary interfaces, premature generics, global state, reflection,
hidden goroutines, unnecessary dependencies, unnecessary codegen.

## Quality Gate (every task, no exceptions)

```
IMPLEMENT
  -> go fmt ./...
  -> go vet ./...
  -> go test ./...
  -> go test -race ./...
  -> benchmark (if performance-relevant, e.g. pkg/orm, pkg/component, pkg/router)
  -> go build ./...           (and `goks build --standalone` for CLI/runtime changes)
  -> security review (see .agents/rules/03-security.md)
  -> public API compatibility review
  -> documentation update (README/API docs/ADR as relevant)
  -> git diff self-review
  -> focused commit(s)
```

For changes touching `pkg/component`, `runtime/client`, or `.gox` compilation:

```
go test -> WASM build -> bundle size check -> hydration/interaction sanity check
```

A task is **not done** just because it compiles. It is done when it passes
this gate.

## Git Discipline

- Never mix unrelated changes in one commit.
- Use branches per feature/fix, never commit directly to `main` for
  non-trivial changes: `feat/...`, `fix/...`, `refactor/...`, `perf/...`,
  `test/...`, `docs/...`.
- Conventional, scoped commit messages, e.g.:
  - `feat(router): add route groups`
  - `fix(action): prevent CSRF bypass`
  - `refactor(component): simplify node reconciliation`
  - `test(orm): add transaction tests`
  - `perf(wasm): reduce hydration payload`
  - `docs(cli): document build pipeline`

## Memory vs. Source of Truth

An agent's persistent memory (preferences, past task notes, learned
shortcuts) is useful context but is **never** authoritative about GoKS's
architecture or decisions. `ARCHITECTURE.md`, `docs/adr/*`, and the code
itself always win over anything remembered from a previous session. If memory
and the repository disagree, trust the repository and correct the memory.

## Task Types and How to Start Them

- **"Audit" tasks** (see `.agents/roadmap/00-foundation.md`): read-only,
  produce a report (e.g. `GOKS_ARCHITECTURE_AUDIT.md`). Do not modify source.
- **"Implement roadmap item X" tasks**: follow the roadmap file for that
  item, create a branch, implement, run the full quality gate, open a PR
  description (even if PRs aren't used yet, write the equivalent summary).
- **"Fix bug" tasks**: reproduce with a failing test first, then fix, keep
  the regression test.
- **Anything architecture-changing**: write or update an ADR under
  `docs/adr/` before writing code.

## Escalation

If a request conflicts with this file, the Constitution, or an accepted ADR,
stop and flag the conflict instead of silently resolving it in favor of the
new request.
