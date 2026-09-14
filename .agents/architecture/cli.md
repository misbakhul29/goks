# Architecture Note: CLI & Tooling (`internal/cli`, `internal/compiler`,
`internal/generator`, `internal/watcher`, `internal/livereload`, `internal/lsp`)

The `goks` CLI: `new`, `dev`, `build`, `start`, `ui`, `generate`, `page`,
`db`, `studio`, `export`, and `lsp`.

Key invariants:
- `goks dev` must hot-reload on file changes (via `internal/watcher`),
  recompile `.gox` -> Go (via `internal/compiler`), recompile WASM, and
  recompile Tailwind, then push updates via `internal/livereload`, without
  requiring a manual restart.
- Compile errors must surface as an in-browser overlay in dev mode, not just
  a terminal stack trace.
- `goks build --standalone` must embed all assets (WASM, CSS, HTML) into a
  single binary with no runtime file dependencies.
- Any file-writing or subprocess-executing CLI code must validate paths and
  avoid path traversal (see `SECURITY.md`).

`internal/lsp` should track `.gox` syntax changes made in
`internal/compiler` — keep them in sync in the same PR when GOX syntax changes.
