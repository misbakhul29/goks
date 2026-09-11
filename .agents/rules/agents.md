---
trigger: always_on
---

# Project Rules & Customizations (`clipper`)

## Git Commit & Versioning Conventions

Every time a new feature is implemented or a major bug is fixed in this repository, follow these rules:

1. **Semantic Versioning & Git Tags**:
   - Always update Semantic Versioning (`vX.Y.Z`):
     - `MINOR` bump (`v1.1.0` -> `v1.2.0`) for new feature implementations.
     - `PATCH` bump (`v1.1.0` -> `v1.1.1`) for major bug fixes.
     - `MAJOR` bump (`v1.0.0` -> `v2.0.0`) for breaking changes.
   - Always create an annotated Git tag matching the version:
     `git tag -a vX.Y.Z -m "vX.Y.Z: <Summary>"`

2. **Workflow Rule**:
   - Proactively assist the user with `git add`, `git commit`, and `git tag` upon finishing feature implementations or major bug fixes.
