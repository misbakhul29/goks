# GoKS Breaking Changes & Migration Guide (v0.15.x to v1.0.0)

This guide documents changes, deprecations, and migrations for applications upgrading from `v0.15.x` to `v1.0.0`.

## Overview of Public API Freeze

Starting with `v1.0.0`, all packages in `pkg/*` are subject to strict Semantic Versioning. Exported identifiers, method signatures, and return contracts will not break during the `v1.x` lifecycle.

---

## 1. Middleware Chain Standardization

### Change:
The router middleware execution order has been formalized and standardized:
1. `Logger`
2. `Recover`
3. `RequestID`
4. `Secure`
5. `MaxBytes`
6. `Timeout`
7. Auth / Application Middlewares

### Migration:
If your application relied on custom middleware ordering, ensure that `router.Recover()` is placed near the outer shell so panics in application handlers are safely caught before propagating to the HTTP listener.

---

## 2. Server Action Origin & Redirect Security

### Change:
- Origin validation in Server Actions now enforces exact-origin matching (`scheme://hostname[:port]`) instead of substring containment.
- Redirects returned from Server Actions must be relative paths (e.g. `/dashboard`); absolute external URLs (`https://evil.com`) and protocol-relative URLs (`//evil.com`) are rejected to prevent open redirect vulnerabilities.

### Migration:
Ensure that server action redirects use relative paths:
```go
// Before:
return action.Redirect("https://example.com/login") // Rejected in v1.0.0

// After:
return action.Redirect("/login")
```

---

## 3. ORM SQL Identifier Validation & Transactions

### Change:
- All table names and column names in `pkg/orm` query builder and CRUD methods are strictly validated against SQL injection attempts. Identifiers containing SQL keywords, semicolons, quotes, or dashes will be rejected.
- `db.Transaction(ctx, fn)` was introduced for atomic operations. If `fn` returns an error or panics, the transaction automatically rolls back.
- Soft-deleted records can now be restored using `orm.Restore(db, model)` and queried using `query.WithTrashed()`.

### Migration:
Update any raw column names passed into `Where` or `OrderBy` to ensure they are valid SQL identifiers.

---

## 4. Universal `pkg/store`

### Change:
- The `pkg/store` reactive state container has been converted into a universal package (no longer restricted by `//go:build js && wasm`). It can now be used and tested on both the server and client.
- The `Subscribe` method returns an unsubscribe function that safely unregisters listeners using unique IDs without slice index shift corruption.

### Migration:
No code changes required. Applications can safely unsubscribe listeners in any order without fear of index corruption.

---

## 5. GOX Diagnostics & Fragment Syntax

### Change:
- The compiler maps syntax and markup errors directly to `.gox` source files with 1-indexed lines, columns, and visual caret snippets.
- Added support for shorthand JSX fragments `<>...</>` and `<Fragment>...</Fragment>`.
- Component props and HTML element attributes are sorted deterministically during compilation.

### Migration:
No code changes required. Existing `.gox` templates compile cleanly with improved error diagnostics.
