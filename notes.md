Active change: replace-make-with-just

Parallelization plan:
- Baseline complete in `openspec/changes/replace-make-with-just/baseline.md`.
- Justfile work complete locally; root Makefile removed.
- Delegated workflow and documentation updates in sub-worktrees, squash-merged locally, and cleaned up.
- Remaining tasks are verification; do not parallelize because commands mutate shared build artifacts (`build/`, `coverage.out`, embedded UI).
- 2026-05-09: Continuing OpenSpec apply for `replace-make-with-just`; no delegation because remaining verification tasks share mutable artifacts.

Review count (post-implementation): 0/5
