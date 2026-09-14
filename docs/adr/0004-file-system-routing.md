# ADR-0004: File-System Routing

Status: Accepted

## Context

Developers coming from Next.js expect predictable, file-based route
definition rather than manually registering every route.

## Decision

`pkg/router` implements Next.js App Router-style file-system routing inside
`app/` (`page.gox`, `layout.gox`), in addition to a programmatic routing API
for cases that need it.

## Consequences

Positive:
- Predictable, discoverable route structure.
- Nested layouts compose naturally with nested directories.

Negative:
- Route-matching precedence (static vs. dynamic vs. catch-all) must be
  explicitly documented and tested to avoid surprising behavior — tracked in
  `.agents/architecture/routing.md`.
