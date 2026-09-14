# ADR-0005: Progressive Server Actions

Status: Accepted

## Context

Modern apps want Next.js-style server functions callable directly from
client code, but GoKS also wants to work with WASM/JS disabled.

## Decision

`pkg/action` implements dual-mode server actions: async `fetch()` via WASM
in modern browsers, with an automatic zero-JS HTML form fallback, and
built-in CSRF protection on both paths.

## Consequences

Positive:
- Actions remain usable without JS/WASM (accessibility, resilience).
- Ergonomic, Next.js-like developer experience for the common case.

Negative:
- Both code paths (WASM `fetch()` and form fallback) must be kept behaviorally
  and security-equivalent, which doubles the surface that needs testing —
  see `.agents/architecture/action.md` and `SECURITY.md`.
