# GoKS Roadmap

Priority levels: **P0** (foundation, blocking), **P1** (core runtime/product),
**P2** (production hardening / ecosystem), **P3** (nice-to-have).

Detail and acceptance criteria for each item live under `.agents/roadmap/`.
This file is the index — keep item names in sync with those files.

The product and engineering execution backlog from v0.15.1 to v1.0.0 lives in
`docs/task/ROAD_TO_V1.md`.

## P0 — Foundation
- Architecture audit (`GOKS_ARCHITECTURE_AUDIT.md`) completed and current.
- Public API surface inventoried and marked stable/experimental.
- Testing infrastructure: `go test ./...` and `go test -race ./...` clean in CI.
- CI pipeline (build, vet, test, race, lint).
- Security baseline reviewed for `pkg/action`, `pkg/auth`, `pkg/rbac`, `pkg/ws`.

## P1 — Runtime
- HTTP lifecycle and middleware stack hardening.
- Routing edge cases (nested layouts, route groups, dynamic segments).
- Context propagation, graceful shutdown.
- Concurrency review of `pkg/ws` hub and `pkg/store`.
- Baseline observability (structured logging).

## P1 — Frontend / WASM
- GOX compiler correctness and error diagnostics.
- Component model + hooks stability (`pkg/component`).
- Selective hydration / islands correctness (0-KB WASM on fully static pages).
- WASM bundle size tracking, TinyGo path (`--compiler=tinygo`) parity.
- Client-side reactive store (`pkg/store`) semantics documented and tested.

## P1 — Data
- ORM query builder coverage and edge cases (`pkg/orm`).
- Migrations workflow.
- Transaction safety, connection pooling.
- Additional database adapters beyond SQLite (evaluate need before building).

## P1 — Developer Experience
- `goks new`, `goks dev`, `goks build`, `goks generate`, `goks ui` polish.
- Compile-error overlay accuracy in `goks dev`.
- `internal/lsp` coverage for `.gox` editing.

## P2 — Production
- Health checks, graceful deployment.
- Metrics/tracing (OpenTelemetry) — only once a concrete need is identified.
- Structured logging consistency across packages.
- Profiling guidance for `goks build --standalone` binaries.

## P2 — Ecosystem
- Plugin/extension points — only with a concrete use case (see anti-over-
  engineering checklist in `AGENTS.md`).
- Additional `goks ui` components.
- Documentation site / expanded guides and migration notes.

## How to Use This Roadmap

1. Pick an item.
2. Read its detail file under `.agents/roadmap/`.
3. If the item implies an architectural decision not yet recorded, write an
   ADR under `docs/adr/` first.
4. Implement following the quality gate in `AGENTS.md`.
5. Update this file's status (e.g. append "(done)") when merged.
