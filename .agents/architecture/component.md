# Architecture Note: Components (`pkg/component`)

Component model and hooks (`UseState`, dynamic children via `component.Any()`)
live here, along with reconciliation logic used both server-side (initial
render) and client-side (WASM hydration/updates).

Key invariants:
- Hooks must preserve call-order semantics across re-renders (same rules as
  React-style hooks: no conditional hook calls).
- Reconciliation must not re-render/re-hydrate a subtree that hasn't
  changed — this is the basis for the "islands" performance story.
- Server-rendered HTML and the first client-side render must produce
  identical DOM structure (no hydration mismatch).

Any change to reconciliation or hook semantics is a fundamental decision —
write an ADR before implementing.
