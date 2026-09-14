# Architecture Note: Server Actions (`pkg/action`, `pkg/rpc`)

Progressive server actions: dual-mode execution — async `fetch()` via WASM
in modern browsers, with automatic zero-JS HTML form fallback, and built-in
CSRF protection.

Key invariants:
- Every action must be reachable and safe via the no-JS form fallback path,
  not only via the WASM/`fetch()` path.
- CSRF tokens are required and validated on both paths identically.
- `pkg/rpc` is the transport between the WASM client and server actions —
  it must not bypass the same validation/auth path that the HTTP form
  submission uses.

Any change here is security-sensitive — see `.agents/rules/03-security.md`
and `SECURITY.md`.
