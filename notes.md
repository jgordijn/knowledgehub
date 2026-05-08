# Daily News Digest Implementation Notes

Review count (post-implementation): 0/5

## Parallelization plan

- Current focus completed: Task group 2.1-2.2 (digest input window/candidate query). These tasks shared the new engine Daily News query code and tests, so they were **not parallelized**.
- Next focus: Task group 2.3-2.4 (scheduling/job lifecycle). This is broad and touches the same scheduler/job state code, so parallelize only after splitting into clearly independent backend/frontend/API areas.
- No delegate sessions launched yet.

## Progress log

- Selected OpenSpec change: `daily-news-digest`.
- Completed task group 1.1-1.4 locally and committed.
- Completed task group 2.1-2.2 locally:
  - Added red tests for previous successful digest lower bounds, failed digest non-advancement, first-run 24-hour fallback, `published_at` matching, and `discovered_at` matching.
  - Implemented `FindDailyNewsCandidates` and `DailyNewsWindow` in `internal/engine`.
  - Added `testutil.CreateSuperuser` and fixed Daily News test collection relations to target the actual `_superusers` collection ID.
- Tests run: `go test ./internal/engine -run TestDailyNews -count=1`.
