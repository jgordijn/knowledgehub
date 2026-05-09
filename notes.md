Active change: replace-make-with-just

Parallelization plan:
- Baseline complete in `openspec/changes/replace-make-with-just/baseline.md`.
- Justfile work complete locally; root Makefile removed.
- Delegated workflow and documentation updates in sub-worktrees, squash-merged locally, and cleaned up.
- Remaining tasks are verification; do not parallelize because commands mutate shared build artifacts (`build/`, `coverage.out`, embedded UI).
- 2026-05-09: Continuing OpenSpec apply for `replace-make-with-just`; no delegation because remaining verification tasks share mutable artifacts.
- 2026-05-09: `just test` passed and printed coverage total (80.3%). Marked task 5.1 complete.
- 2026-05-09: `just build` passed; confirmed `build/knowledgehub` executable and embedded `cmd/knowledgehub/ui/build/index.html`. Marked task 5.2 complete.
- 2026-05-09: `just release` and `just release v9.8.7` passed. Tarball contents: `knowledgehub`, service, updater script/service/timer. Marked task 5.3 complete.

Review count (post-implementation): 0/5
