# Contributing to GoKS

Thank you for considering a contribution. GoKS aims to be a coherent,
production-grade fullstack Go framework — please read this before opening a
PR, and see `AGENTS.md` if you are an AI agent.

## Before You Start

1. Read `GOKS_CONSTITUTION.md` and `ARCHITECTURE.md`.
2. For anything touching public APIs (`pkg/*` exported identifiers) or core
   architecture, check `docs/adr/` first — there may already be a decision on
   record, or you may need to write a new ADR.
3. Search existing issues/roadmap items in `ROADMAP.md` before proposing a
   new abstraction.

## Development Setup

```bash
git clone https://github.com/misbakhul29/goks.git
cd goks
go mod tidy
go build ./...
go test ./...
```

## Branching & Commits

- One branch per feature/fix: `feat/...`, `fix/...`, `refactor/...`,
  `perf/...`, `test/...`, `docs/...`.
- One logical change per commit. Conventional, scoped messages:
  `feat(router): add route groups`, `fix(action): prevent CSRF bypass`, etc.

## Quality Gate (required before requesting review)

```
go fmt ./...
go vet ./...
go test ./...
go test -race ./...
go build ./...
```

For WASM/`.gox`-affecting changes, also verify the WASM build and check
bundle size impact.

## Pull Request Checklist

- [ ] Tests added/updated
- [ ] `go vet` and `go test -race ./...` pass
- [ ] Public API changes documented (and an ADR added if breaking)
- [ ] Security implications considered (see `.agents/rules/03-security.md`)
- [ ] Docs (`README.md` / `ARCHITECTURE.md` / relevant guide) updated if
      behavior changed

## Code Style

- Idiomatic Go, `gofmt`-clean.
- Prefer explicit code over generics/reflection/interfaces unless justified
  by a concrete current use case (see the anti-over-engineering checklist in
  `AGENTS.md`).
- No Node.js/npm dependency may be introduced.
