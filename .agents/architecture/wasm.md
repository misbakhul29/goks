# Architecture Note: WASM / Islands (`runtime/client`, `pkg/component`, TinyGo path)

GoKS uses selective hydration ("islands"): static pages ship as pure HTML
with no WASM; pages with interactive components load a WASM bundle scoped to
just that interactivity. See `docs/adr/0002-wasm-islands.md`.

Key invariants:
- Static-only pages: 0 KB WASM.
- `--compiler=tinygo` build path must stay functional and produce
  meaningfully smaller bundles (<300 KB target) than the standard `go`
  WASM build.
- Any new client-side API must work through the existing hydration bootstrap
  rather than introducing a second bootstrap mechanism.

Track bundle size impact for every change here (see
`.agents/rules/05-performance.md`).
