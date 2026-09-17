---
trigger: always_on
---

# Git Rules

- Never mix unrelated changes in one commit.
- One branch per feature/fix: `feat/...`, `fix/...`, `refactor/...`,
  `perf/...`, `test/...`, `docs/...`.
- Conventional, scoped commit messages, e.g.:
  - `feat(router): add route groups`
  - `fix(action): prevent CSRF bypass`
  - `refactor(component): simplify node reconciliation`
  - `test(orm): add transaction tests`
  - `perf(wasm): reduce hydration payload`
  - `docs(cli): document build pipeline`
- Do not commit directly to `main` for non-trivial changes — open a branch,
  self-review the diff against the quality gate, then merge.
- Semantic-version tags (`vX.Y.Z`) are reserved for milestone releases (not per-commit):
  MINOR for new features, PATCH for batched bug fixes, MAJOR for breaking changes.
  Only create tags when explicitly requested or cutting a release.
