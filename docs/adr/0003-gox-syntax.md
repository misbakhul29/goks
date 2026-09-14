# ADR-0003: GOX Declarative Markup Syntax

Status: Accepted

## Context

Writing UI directly with `html/template` or manual string building is
verbose and error-prone. GoKS wants a declarative, JSX-like authoring
experience while staying 100% Go underneath.

## Decision

Introduce `.gox` files with a declarative, JSX-like syntax
(`<div><h1>{title}</h1></div>`, `<ui.Button />`) that compiles directly into
plain Go code (`internal/compiler`, `pkg/compiler`) in an isolated build
workspace, rather than being interpreted at runtime.

## Consequences

Positive:
- Familiar, ergonomic authoring for developers used to JSX-like syntax.
- Compiles to plain Go — no runtime template interpretation overhead.

Negative:
- Requires a custom compiler and editor tooling (`internal/lsp`) to keep in
  sync with syntax changes.
- Compile error messages must be mapped back to `.gox` source locations to
  stay developer-friendly (tracked in `.agents/roadmap/02-frontend.md`).
