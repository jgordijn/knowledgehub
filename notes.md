# Daily News Digest Implementation Notes

Review count (post-implementation): 0/5

## Parallelization plan

- Current focus completed: Task group 2.1-2.2 (digest input window/candidate query). These tasks shared the new engine Daily News query code and tests, so they were **not parallelized**.
- Current focus completed: Task group 2.3-2.4 (scheduling/job lifecycle). These tasks both touched `internal/engine/daily_news_scheduler.go`, `internal/engine/scheduler.go`, and lifecycle tests, so they were **not parallelized**.
- Next focus: Task group 3.1-3.5 (AI digest generation). Prompt construction and response parsing may be separable only after the generator interface is stable; when in doubt, do not parallelize.
- No delegate sessions launched yet.

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
