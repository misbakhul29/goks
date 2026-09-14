# Roadmap Detail: P1 — Developer Experience / Tooling

## Goal
Keep `goks new` / `goks dev` / `goks build` / `goks generate` / `goks ui`
fast, reliable, and pleasant.

## Items
- Verify `goks dev` hot-reload latency and correctness across file types
  (`.gox`, Go, Tailwind config).
- Confirm the in-browser compile-error overlay covers GOX compile errors,
  Go build errors, and Tailwind errors distinctly.
- `internal/lsp` coverage check against current GOX syntax (keep in sync
  with `pkg/compiler`/`internal/compiler`).
- `goks ui add` component coverage and consistency with Tailwind v4.

## Acceptance Criteria
- Documented, current list of `goks` CLI commands and flags in
  `.agents/architecture/cli.md` matching actual `internal/cli` code.
