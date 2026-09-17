# Architecture Note: Routing (`pkg/router`)

File-system routing modeled on Next.js's App Router (`app/page.gox`,
`app/layout.gox`), plus a thread-safe programmatic routing API.

Key invariants:
- Thread safety: `Router` operations (route registration, group creation,
  middleware assignment, and `ServeHTTP` dispatch) are protected by a
  `sync.RWMutex` to prevent data races during runtime route introspection or
  dynamic registration.
- Route matching precedence: static (100) > dynamic (10) > wildcard (1)
  precedence is calculated per segment to resolve ambiguous matches deterministically.
- Nested route groups: `Group.Group(prefix, mw...)` creates sub-groups with
  chained prefixes and inherited middlewares.
- Layouts must compose (nested layouts wrap child routes) without duplicating
  render work.
- Route groups and path parameters must be exposed to both server render and
  server actions consistently.
