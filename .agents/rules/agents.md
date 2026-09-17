---
trigger: always_on
---

# Project Rules & Customizations

## Git Commit & Release Conventions

Follow these rules for development, commits, and releases:

1. **Commit Discipline (Routine Tasks)**:
   - Use clean, conventional scoped commit messages: `feat(...)`, `fix(...)`, `refactor(...)`, `perf(...)`, `test(...)`, `docs(...)`.
   - Proactively assist the user with `git add` and `git commit` for completed tasks.
   - **Do NOT automatically bump versions or create Git tags on every task/commit.** Commits are the unit of daily work, while tags are milestone releases.

2. **Milestone Release & Git Tags (On-Demand Only)**:
   - Only create Git tags when the user **explicitly asks** to release a new version or cut a milestone.
   - Batch multiple bug fixes and enhancements into a single release rather than releasing a tag per commit.
   - When a release is requested:
     - Update Semantic Versioning in `internal/version/version.go`:
       - `PATCH` bump (`v1.4.0` -> `v1.4.1`): Bundled bug fixes and maintenance improvements.
       - `MINOR` bump (`v1.4.0` -> `v1.5.0`): New features and backward-compatible enhancements.
       - `MAJOR` bump (`v1.0.0` -> `v2.0.0`): Breaking changes to public API (requires ADR).
     - Create an annotated Git tag matching the milestone:
       - Framework: `git tag -a vX.Y.Z -m "vX.Y.Z: <Summary>"`
       - VS Code Extension: `git tag -a editors/vscode/vX.Y.Z -m "editors/vscode/vX.Y.Z: <Summary>"`