# GoKS Constitution

## Vision

GoKS is a production-grade fullstack web framework where developers build
modern, interactive web applications using **Go as the only application
language** — backend APIs, server-rendered HTML, and client-side
interactivity (via selective WebAssembly hydration) — shipping as a single
standalone binary, with zero required Node.js/npm dependency.

## Non-Negotiables

1. Go-first — no required JS/Node.js runtime dependency.
2. Production-ready — every merged feature must be usable in production, not
   a demo.
3. Type-safe — leverage Go's type system end-to-end (server, ORM, actions,
   client state).
4. Secure-by-default — CSRF, XSS, session, and auth protections are on
   unless explicitly and knowingly disabled.
5. Observable — logging, and eventually metrics/tracing, are first-class.
6. Testable — every package should be unit-testable without a browser or
   real database where possible.
7. Scalable — the runtime must support real concurrent workloads without
   architectural rewrites.
8. Maintainable — code should be understandable by a new contributor reading
   `ARCHITECTURE.md` plus the package's own tests.
9. Minimal unnecessary dependencies — every new dependency is a liability;
   justify it.
10. Excellent developer experience — `goks new`, `goks dev`, `goks build`
    should stay fast and pleasant.

## Architectural Priorities (tie-breaker order)

```
Correctness > Security > API stability > Maintainability > Performance
> Developer experience > Feature velocity
```

When two goals conflict, the one listed first wins, and the trade-off must be
written down (in the PR description, commit message, or an ADR).

## What GoKS Is Not

- Not a JavaScript framework with a Go backend bolted on.
- Not a place for speculative, unused abstractions "for future flexibility."
- Not a collection of independent features (router + ORM + auth + websocket +
  wasm + compiler + CLI) — it is one coherent platform. Every new package
  must justify how it fits the platform, not just "adds a feature."

## Amending This Document

Changes to the Constitution itself require an ADR explaining why a
non-negotiable or priority ordering is being changed, plus the migration
impact on existing GoKS applications.
