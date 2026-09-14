# Architecture Note: Routing (`pkg/router`)

File-system routing modeled on Next.js's App Router (`app/page.gox`,
`app/layout.gox`), plus a programmatic routing API.

Key invariants:
- Route matching must be deterministic and documented (static > dynamic >
  catch-all precedence, or whatever precedence is currently implemented —
  document it explicitly here once confirmed against code).
- Layouts must compose (nested layouts wrap child routes) without duplicating
  render work.
- Route groups/params must be exposed to both server render and server
  actions consistently.

Update this file with the actual, current precedence rules the first time an
agent verifies them against `pkg/router`'s implementation — do not leave this
as a placeholder once verified.
