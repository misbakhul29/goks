# Roadmap Detail: P1 — Frontend / WASM

## Goal
Make the GOX + component + hydration story rock-solid and measurable.

## Items
- GOX compiler error diagnostics: point to the exact `.gox` source location,
  not just the generated Go location.
- Verify hook call-order invariants in `pkg/component` are enforced (fail
  loudly, not silently, on violation).
- Confirm 0-KB WASM on fully static pages with an automated bundle-size
  check.
- TinyGo path (`--compiler=tinygo`) parity: same feature set as the standard
  `go` WASM build, tracked bundle size.
- Document `pkg/store` reactivity semantics (what triggers a re-render,
  batching behavior, if any).

## Acceptance Criteria
- Bundle-size regression check wired into CI (even a simple size-diff
  report).
- `.agents/architecture/wasm.md` and `component.md` updated with verified
  (not placeholder) behavior.
