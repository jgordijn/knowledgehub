# Working Notes

Review count (post-implementation): 4/5

## Current pass

Using change: daily-news-digest. Override by selecting another OpenSpec change explicitly.
OpenSpec tasks are 37/37 complete, but review 4 found implementation gaps to fix before final handoff.

Parallelization check: not delegating this pass. The remaining fixes are tightly coupled in Daily News routes/engine tests and coverage, so parallel sub-worktrees would risk clashing.

Planned small TDD steps:
1. Add failing route test for `GET /api/daily-news/digests/{id}` owner-scoped detail DTO.
2. Implement digest detail route.
3. Add failing worker/engine test for clear missing OpenRouter API key failure.
4. Implement user-safe missing configuration error preservation.
5. Run targeted tests/coverage, then broader verification.

## Daily News review report

Objective check: `objective.md` is not present in this worktree, so this review used the Daily News OpenSpec proposal/design/spec (`openspec/changes/daily-news-digest/`) as the objective. Verdict: not ready for proof; developer fixes are needed.

Findings:

1. Missing required digest detail read route.
   - Objective/design requires `GET /api/daily-news/digests/{id}` to return a caller-owned digest DTO or an auth-safe not-found/denied response (`openspec/changes/daily-news-digest/design.md:130`).
   - Registered routes currently include list, generate, regenerate, and entry-reference routes only; there is no direct digest detail route (`internal/routes/daily_news.go:85-145`).
   - Impact: clients cannot use the specified route-level digest read contract; tests also do not cover it.

2. Missing AI configuration failures are not clear to users.
   - The worker detects missing OpenRouter API key and passes `"OpenRouter API key is not configured."` (`internal/engine/daily_news_scheduler.go:268`), but `sanitizeDailyNewsError` discards all non-empty messages and always stores `"Digest generation failed. Please try again."` (`internal/engine/daily_news_scheduler.go:440-445`).
   - Objective requires a clear, user-safe missing-configuration failed state.
   - Impact: users cannot tell they need to configure OpenRouter, and the settings/remediation path is hidden.

3. Test coverage and scenario coverage are below the project target.
   - Coverage run: `go test ./internal/ai/... ./internal/engine/... ./internal/routes/... -coverprofile=.tmp/review-cover.out -covermode=atomic && go tool cover -func=.tmp/review-cover.out | tail -1` => total 87.3% (routes 80.6%), below the documented >90% target and below the agent instruction to fully cover created logic.
   - Missing coverage aligns with the issues above: no registered-route test for `GET /api/daily-news/digests/{id}` and no worker/route test proving missing AI configuration stores a clear sanitized error.

Verification performed:
- `go test ./... -count=1` passed.
- `cd ui && bunx vitest run` passed.
- `openspec validate daily-news-digest --strict` passed.
- `go vet ./...` passed.
- `make build` passed; existing Svelte a11y warnings remain. Restored `cmd/knowledgehub/ui/build/.gitkeep` after the build.

## Fix pass results

Completed local TDD fixes without parallel delegation:
- Added red/green handler test for owner-scoped `HandleDailyNewsGetDigest`.
- Implemented `GET /api/daily-news/digests/{id}` registration and handler with route-level owner enforcement.
- Added registered-route integration coverage for digest detail reads.
- Added red/green worker test for missing OpenRouter API key failures.
- Preserved a clear, user-safe missing API key digest error while continuing to sanitize other provider/internal errors.

Verification:
- `go test ./internal/routes -run TestHandleDailyNewsGetDigestReturnsOwnedDigestAndDeniesCrossUser -count=1` passed.
- `go test ./internal/engine -run TestProcessPendingDailyNewsJobsStoresClearMissingAPIKeyFailure -count=1` passed.
- `go test ./internal/routes -run TestDailyNewsRegisteredDigestDetailRouteReturnsOwnedDigest -count=1` passed.
- `go test ./internal/ai/... ./internal/engine/... ./internal/routes/... -coverprofile=.tmp/cover.out -covermode=atomic` passed; total coverage now 87.9% (up from review's 87.3%).
- `go test ./... -count=1` passed.
- `openspec validate daily-news-digest --strict` passed.

OpenSpec tasks remain 37/37 checked complete in `openspec/changes/daily-news-digest/tasks.md`.
