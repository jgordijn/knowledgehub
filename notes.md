# Review report — Daily News Digest

Review count (post-implementation): 6/5 (additional review explicitly requested by Jeroen)

## Verification run

- `go test ./... -count=1` — passed
- `cd ui && bunx vitest run` — passed
- `openspec validate daily-news-digest --strict` — passed
- `go test ./internal/ai/... ./internal/engine/... ./internal/routes/... -coverprofile=.tmp/review-cover.out -covermode=atomic` — passed, total 87.9%
- `go vet ./...` — passed
- `make build` — passed; existing Svelte a11y warnings remain. Restored `cmd/knowledgehub/ui/build/.gitkeep` after build.

`objective.md` is not present in this worktree, so objective review was performed against `openspec/changes/daily-news-digest/proposal.md`, `design.md`, and `specs/daily-news/spec.md`.

## Objective assessment

Daily News is broadly implemented: backend collections/routes/scheduler/generator, frontend navigation/page/rendering/settings/archive/modal, and OpenSpec validation are present. However the objective is not fully proven/complete because the review found functional idempotency/API-contract gaps and incomplete tests for the actual frontend page plus created-code coverage below the project requirement.

## Review comments that need DEVELOPER attention

1. **Pre-due Generate now is not idempotent while an active manual job exists.**
   - `internal/routes/daily_news.go:297` uses `now.UTC().Truncate(time.Second)` as the pre-due manual `periodEnd`.
   - `internal/engine/daily_news_scheduler.go:130-141` builds/reuses active jobs only by exact `(user, local_date, period_start, period_end)` window key.
   - Result: two pre-due Generate now requests seconds apart can create two active manual digest jobs for the same user/local day while the first is still `pending`/`running`. This weakens the proposal's idempotent manual generation and deterministic active-job locking objective. Add a failing route/scheduler test for repeated pre-due Generate now with different `now` seconds, then fix by canonicalizing/reusing an active manual job for the intended window/day.

2. **Generate now can return `200 OK` for an active scheduled regeneration instead of the active `202 Accepted` state.**
   - `internal/engine/daily_news_scheduler.go:135` checks `successful_scheduled_day_key` before `active_scheduled_day_key`.
   - `internal/routes/daily_news.go:315-316` maps that error to `findSuccessfulScheduledDigest`, and `internal/routes/daily_news.go:397-399` does not filter out active statuses.
   - A previously successful scheduled digest being regenerated keeps `successful_scheduled_day_key` while status becomes `pending`; Generate now for that date can therefore return `200 OK` for a `pending` record instead of reusing the active job with `202 Accepted`. Add a regression test and prefer active scheduled locks/status before treating the day as already successfully complete.

3. **Frontend tests do not really exercise the Daily News page functionality marked complete in tasks.**
   - `ui/src/lib/daily-news-ui.test.ts` covers pure helpers and sanitizer behavior only.
   - `rg` found no tests importing/rendering `ui/src/routes/daily-news/+page.svelte` or `ui/src/lib/components/DailyNewsDigest.svelte`, and no tests driving `generateNow`, `regenerateDigest`, `saveSettings`, archive load-more/selection, polling, or reference-modal interactions.
   - This leaves tasks 5.1/5.6/6.3/7.2/8.2 under-proven. Add Svelte component/integration tests with mocked `pb.send` and user interactions so the tests verify the UI behavior, not only helper functions.

4. **Created-code test coverage is below the project requirement.**
   - Targeted package coverage is 87.9%, and new Daily News functions still have uncovered branches, e.g. `selectDailyNewsPromptCandidates` 66.7%, `formatTakeaways` 28.6%, `RunDailyNewsSchedule` 70.4%, `HandleDailyNewsRegenerate` 78.8%, `RegisterDailyNewsRoutes` 40.7%.
   - The project instruction requires created code to have 100% coverage and tests to cover logic paths. Add focused backend tests for the uncovered error/idempotency branches and frontend component tests above.

## Other review notes

- No new blocking XSS issue was found in the Daily News Markdown path: raw HTML/images/dangerous schemes are covered by helper tests and DOMPurify allowlisting.
- Code clarity is generally acceptable, but the three separate `getOrCreateDailyNewsSettings` implementations in `cmd/knowledgehub`, `internal/engine`, and `internal/routes` are duplication and should be considered for consolidation after the functional/test fixes.
- No obsolete feature code was identified beyond that helper duplication.

## Developer follow-up 2026-05-09

Using change: daily-news-digest (override with /opsx-apply <other>). OpenSpec tasks are all checked off; implementing reviewer follow-up defects before handoff.

Parallelization plan:
- Coordinator owns backend idempotency/API-contract fixes and shared notes/tasks.
- Delegate owns frontend component/integration test coverage only under ui/src/** test files. Forbidden: backend files, OpenSpec tasks, notes.md.
- No parallelization for backend review items 1 and 2 because they touch the same route/scheduler code paths.

Review count (post-implementation): 6/5; no more post-implementation review loops will be requested.

Backend follow-up progress:
- Added failing route regressions for pre-due manual Generate now across different seconds and active scheduled regeneration response status.
- Fixed ClaimDailyNewsJob to prefer active scheduled locks before successful scheduled-day reservations.
- Fixed pre-due manual Generate now to reuse active manual jobs for the same user/local date before building a new second-specific window.
- Verification: go test ./internal/routes ./internal/engine -run 'TestHandleDailyNewsGenerateNow(ReusesPreDueManualJobAcrossSeconds|ReturnsActiveScheduledRegeneration)|TestClaimDailyNewsJob|TestRunDailyNewsSchedule' -count=1 passed.
