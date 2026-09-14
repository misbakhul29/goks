# Roadmap Detail: P0 — Foundation

## Goal
Establish ground truth about the current state of GoKS before adding more
features.

## Items

### Architecture audit
Produce `GOKS_ARCHITECTURE_AUDIT.md` at repo root. This is a **read-only,
discovery-only** task — do not modify source code while doing it.

Required sections: current architecture, strengths, weaknesses, critical
risks, technical debt, API stability risks, performance risks, security
risks, testing gaps, developer-experience gaps, scalability risks,
recommended architecture, migration strategy, prioritized roadmap.

### Public API inventory
List every exported identifier in `pkg/*`, mark each `stable` or
`experimental`. Anything not explicitly marked stable can change without an
ADR; anything marked stable follows the API-stability rule in `AGENTS.md`.

### Testing infrastructure
Get `go test ./...` and `go test -race ./...` passing cleanly across the
whole repo. Fix or explicitly document (don't silently skip) any currently
failing/flaky test.

### CI pipeline
Set up CI to run: `go fmt -l .` (fail on unformatted files), `go vet ./...`,
`go test ./...`, `go test -race ./...`, `go build ./...`.

### Security baseline
Walk `SECURITY.md`'s checklist against `pkg/action`, `pkg/auth`, `pkg/rbac`,
`pkg/ws` as they exist today; file findings as issues, do not fix silently in
the same audit pass unless trivial and clearly in-scope.

## Acceptance Criteria
- `GOKS_ARCHITECTURE_AUDIT.md` exists and matches the current code.
- CI is green on `main`.
- Security findings are documented, even if not yet all fixed.
