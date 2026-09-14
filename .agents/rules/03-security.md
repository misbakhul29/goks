---
trigger: always_on
---

# Security Rules

Every change touching `pkg/action`, `pkg/auth`, `pkg/rbac`, `pkg/router`,
`pkg/ws`, or file/process handling in `internal/cli` must consider, at
minimum: authentication, authorization, CSRF, XSS, SSRF, path traversal,
request size/timeouts, session handling, and secret handling. See
`SECURITY.md` for the full checklist.

Never weaken an existing security control (CSRF token check, RBAC check,
input validation, request limits) to make a feature easier to implement or a
test easier to pass. If a control is genuinely blocking legitimate work,
raise it explicitly rather than removing it quietly.
