# Daily News Digest Implementation Notes

Review count (post-implementation): 0/5

## Parallelization plan

- Current focus completed: Task group 2.1-2.2 (digest input window/candidate query). These tasks shared the new engine Daily News query code and tests, so they were **not parallelized**.
- Current focus completed: Task group 2.3-2.4 (scheduling/job lifecycle). These tasks both touched `internal/engine/daily_news_scheduler.go`, `internal/engine/scheduler.go`, and lifecycle tests, so they were **not parallelized**.
- Current focus completed: Task group 3.1-3.5 (AI digest generation). Prompt construction, generator interface, response parsing, failed-state recording, and empty-window handling all shared the same AI/engine integration surface and tests, so they were **not parallelized**.
- Current focus completed: Task group 4.1-4.2 (manual Generate now API). Tests and implementation shared the new route handler, auth-derived owner behavior, and engine claim path, so they were **not parallelized**.
- Current focus completed: Task group 4.3-4.4 (manual Regenerate API). Tests and implementation both touched `internal/routes/daily_news.go`, `internal/routes/daily_news_test.go`, and Daily News job lifecycle behavior, so they were **not parallelized**.
- Current focus completed: Task 4.5 concurrency/lock tests. This extended existing database uniqueness/lock coverage in `internal/engine/daily_news_scheduler_test.go`; it was **not parallelized**.
- Current focus completed: frontend task group 5.1-5.2. These touched `ui/src/lib/components/Sidebar.svelte`, a new route, and small UI helpers/tests, so they were done locally and **not parallelized**.
- Current focus completed: task 5.3 latest digest display/sanitizer. This owned the Daily News UI rendering/sanitizer boundary (`ui/src/lib/daily-news-ui.*`, new Daily News component, and `/daily-news` route), so it was **not parallelized** with 5.4-5.6 or 6.x reference rendering.
- Current focus: task 5.4 pending/failed/empty UI states. This extends the same Daily News route/component state rendering from 5.3, so it is **not parallelized**.
- Checked remaining tasks for safe delegation after 5.2: 6.x and 7.x need backend route contracts; no delegate session launched yet.

## Progress log

- Selected OpenSpec change: `daily-news-digest`.
- Completed task group 1.1-1.4 locally and committed.
- Completed task group 2.1-2.2 locally:
  - Added red tests for previous successful digest lower bounds, failed digest non-advancement, first-run 24-hour fallback, `published_at` matching, and `discovered_at` matching.
  - Implemented `FindDailyNewsCandidates` and `DailyNewsWindow` in `internal/engine`.
  - Added `testutil.CreateSuperuser` and fixed Daily News test collection relations to target the actual `_superusers` collection ID.
- Tests run: `go test ./internal/engine -run TestDailyNews -count=1`.
- Completed task group 2.3-2.4 locally:
  - Added red/green tests for schedule validation, local timezone due checks, disabled settings, same-day catch-up, DST edges, deterministic active job keys, duplicate active/success prevention, pending-to-running-to-terminal transitions, stale active-job recovery, failed retry, and scheduler settings scans.
  - Implemented Daily News schedule validation/due logic, deterministic job claim keys, pending claim/terminal completion helpers, stale recovery, schedule scanning, and scheduler hook-in.
- Tests run: `go test ./internal/engine -run 'TestDailyNews|TestRunDailyNews|TestScheduler' -count=1`.
- Completed task 3.1 locally:
  - Added red/green prompt construction coverage for summaries, takeaways, stars, source labels, dates, IDs, bounded/delimited extra instructions, prompt-injection boundaries, deterministic capping, and candidate/included metadata.
- Tests run: `go test ./internal/engine -run TestBuildDailyNewsPrompt -count=1`.
- Completed task group 3.2-3.5 locally:
  - Added red/green tests for structured JSON generation, invalid-reference filtering, duplicate reference deduplication, malformed AI response errors, sanitized failed-state recording, and successful empty-window "No articles today" output.
  - Implemented Daily News AI completion wrapper use, response parsing/validation against included candidate IDs, empty-window digest result, and failure recording via existing sanitized terminal job helper.
- Tests run: `go test ./internal/ai ./internal/engine -run 'TestGenerateDailyNewsDigest|TestRecordDailyNewsFailure|TestBuildDailyNewsPrompt|TestSetCompleteFunc' -count=1`.
- Completed task group 4.1-4.2 locally:
  - Added red/green route-handler tests for authenticated Generate now queuing (`202 Accepted`), persisted pending jobs, active job reuse, same-day scheduled success idempotency (`200 OK`), failed retry, owner scoping, and unauthenticated denial.
  - Implemented `POST /api/daily-news/generate` registration plus `HandleDailyNewsGenerateNow`, deriving ownership from auth/user ID, materializing per-user default settings, canonical scheduled/manual window derivation, and reusing the existing Daily News claim path.
- Tests run: `go test ./internal/routes -run TestHandleDailyNewsGenerateNow -count=1`.
- Tests run: `go test ./internal/engine ./internal/routes -run 'TestDailyNews|TestHandleDailyNewsGenerateNow|TestRunDailyNews|TestScheduler' -count=1`.
- Completed task group 4.3-4.4 locally:
  - Added red/green route and lifecycle tests for owner-scoped Regenerate, unauthenticated/cross-user denial, selected active digest reuse, successful-snapshot preservation while active, successful content replacement, sanitized failed regeneration state, and scheduled success reservation preservation.
  - Implemented `POST /api/daily-news/digests/{id}/regenerate`, `HandleDailyNewsRegenerate`, `CompleteDailyNewsRegeneration`, and `FailDailyNewsRegeneration`.
- Tests run: `go test ./internal/routes -run 'TestHandleDailyNewsRegenerate|TestCompleteDailyNewsRegeneration' -count=1`.
- Tests run: `go test ./internal/engine ./internal/routes -run 'TestDailyNews|TestHandleDailyNewsGenerateNow|TestHandleDailyNewsRegenerate|TestCompleteDailyNewsRegeneration|TestRunDailyNews|TestScheduler' -count=1`.
- Completed task 4.5 locally:
  - Added tests proving canonical lock reuse for scheduled/manual same-window races with sub-second `now` differences, concrete SQLite uniqueness for non-empty `active_window_key`, and pre-due manual success not suppressing a later scheduled claim.
- Tests run: `go test ./internal/engine -run 'TestDailyNewsConcreteLockIndexes|TestDailyNewsPreDueManual' -count=1`.
- Completed task group 5.1-5.2 locally:
  - Added red/green Vitest coverage for the Daily News nav item contract and page loading message.
  - Added `daily-news-ui` helpers, a Daily News sidebar navigation item, and `/daily-news` route with initial loading-state page shell.
- Tests run: `cd ui && bunx vitest run src/lib/daily-news-ui.test.ts`.
- Check attempted: `cd ui && bun run check` currently fails on pre-existing TypeScript/Svelte issues in `vite.config.ts`, `LinkPanel.svelte`, and `QuickAddModal.svelte`; no new Daily News diagnostics were reported.
- Started task 5.3 locally: no safe parallel slice identified because sanitizer policy, latest digest DTO shape, and route rendering all share the same files/contracts.
- Completed task 5.3 locally:
  - Added red/green tests for strict Daily News Markdown sanitization and subset indication.
  - Added `renderDailyNewsMarkdown`, Daily News DTO/subset helpers, `DailyNewsDigest.svelte`, and latest digest rendering from the Daily News route DTO.
  - Marked OpenSpec task 5.3 complete.
- Tests run: `cd ui && bunx vitest run src/lib/daily-news-ui.test.ts`.
- Check attempted: `cd ui && bun run check` still fails on pre-existing diagnostics in `vite.config.ts`, `LinkPanel.svelte`, `QuickAddModal.svelte`, and existing a11y warnings; no Daily News diagnostics were reported.
- Started task 5.4 locally: no safe parallel slice identified because the state messaging belongs in the same Daily News UI helpers/page contract.
