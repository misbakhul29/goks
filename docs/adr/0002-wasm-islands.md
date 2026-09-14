# ADR-0002: Selective WASM Hydration ("Islands")

Status: Accepted

## Context

GoKS needs client-side interactivity without requiring a JavaScript runtime,
while keeping static pages fast and lightweight.

## Decision

Use selective WASM hydration. Static pages must not load WASM. Interactive
components are compiled into WASM "islands" and hydrated only where needed
(`component.ClientBase` or event handlers).

## Consequences

Positive:
- Zero JS runtime; Go-only application code end-to-end.
- Smaller static pages (0 KB WASM where no interactivity exists).

Negative:
- WASM compilation cost during build.
- Browser interoperability complexity (bundle size, load time) — mitigated
  by the TinyGo compile path (`--compiler=tinygo`, <300 KB target).
- Requires careful hydration-boundary detection to avoid over-hydrating
  pages.
