---
trigger: always_on
---

# Workspace Rules

1. Always read `AGENTS.md` at the repository root before starting any task.
2. Always read every file in this `.agents/rules/` directory before starting
   any task.
3. Treat `ARCHITECTURE.md`, `docs/adr/*`, and the code itself as source of
   truth — never your own memory of a previous session.
4. Never modify `main`/`master` directly for non-trivial changes; use a
   feature branch.
5. Never mark a task complete without running the full quality gate defined
   in `AGENTS.md`.
6. If a request conflicts with these rules, the Constitution, or an accepted
   ADR, stop and flag the conflict instead of silently proceeding.
