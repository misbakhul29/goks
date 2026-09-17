# Architecture Note: Server Actions (`pkg/action`, `pkg/rpc`)

Progressive server actions: dual-mode execution (async `fetch()` via WASM
in modern browsers, with automatic zero-JS HTML form fallback, and built-in
CSRF protection).

Key invariants:
- Every action must be reachable and safe via the no-JS form fallback path,
  not only via the WASM/`fetch()` path.
- In fallback redirects, same-origin Referer paths and query strings must be
  preserved (`target.RequestURI()`), while external hosts and protocol-relative
  URIs (`//evil.com`, `/\evil.com`) must be sanitized to `/` to prevent open-redirect vulnerabilities.
- CSRF validation rejects `Sec-Fetch-Site: cross-site` requests immediately.
  Origin and Referer headers must match the request scheme and host.
- When `action.SetSecret(...)` is configured, cryptographic HMAC-SHA256 tokens
  (`_csrf` field or `X-CSRF-Token` header) are strictly validated.
- Helper `action.Form(actionName, props, children...)` provides a unified
  component representation that automatically injects POST endpoint, `_action`,
  and `_csrf` tokens.
- `Context.Get` and `Context.GetAll` transparently support both standard
  urlencoded forms and multipart form payloads. File uploads are accessed via
  `Context.File(key)`.
- `pkg/rpc` is the transport between the WASM client and server actions:
  it must not bypass the same validation/auth path that the HTTP form
  submission uses.

Any change here is security-sensitive: see `.agents/rules/03-security.md`
and `SECURITY.md`.
