# ADR-0007: Reactive Global Store for Client State

Status: Accepted

## Context

Interactive WASM components sometimes need to share state across component
boundaries (cross-component reactive synchronization), beyond local
`UseState`.

## Decision

Provide `pkg/store` (`store.New()`) as a reactive global store for
cross-component state synchronization in the client WASM runtime.

## Consequences

Positive:
- Enables cross-component state sharing without prop-drilling.

Negative:
- Global mutable state is a well-known source of bugs; usage guidance and
  clear reactivity semantics (what triggers a re-render, batching) must be
  documented — tracked in `.agents/architecture/component.md` and roadmap
  item `.agents/roadmap/02-frontend.md`.
