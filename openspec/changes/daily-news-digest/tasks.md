## 1. Data Model and Test Fixtures

- [x] 1.1 Add failing tests for `daily_news_settings` and `daily_digests` collection creation, defense-in-depth owner-scoped auth rules, read-only user-facing digest collection access, denied generic settings create/delete, persisted defaults, one-settings-record-per-user uniqueness, explicit server-route owner enforcement for `_superusers`, and user ownership.
- [x] 1.2 Implement PocketBase collections for Daily News settings and digests, including a unique settings user index, digest trigger/concrete SQLite-backed lock-key/snapshot/attempt/heartbeat fields, non-empty active/success lock unique indexes, and denying generic user-facing digest create/update/delete rules.
- [x] 1.3 Add testutil helpers for creating Daily News settings and digest records.
- [x] 1.4 Add migration/backfill behavior or startup defaults by enumerating PocketBase `_superusers`, including users created after startup, with idempotent get-or-create/upsert behavior.

## 2. Digest Window and Scheduling Logic

- [x] 2.1 Add failing tests for digest input window selection: previous successful digest, failed digest non-advancement, first 24-hour fallback, published_at match, and discovered_at match.
- [x] 2.2 Implement digest candidate query logic using entries visible to the target user.
- [x] 2.3 Add failing tests for timezone due checks, same-day missed-run catch-up after downtime, no automatic previous-day backfill, invalid timezone/time rejection, disabled settings, duplicate same-local-day prevention, active `pending -> running -> success|failed` status transitions, stale pending/running job recovery after crash/redeploy, failed retry behavior, deterministic canonical job/window keys, pre-due manual generation not suppressing the later scheduled digest, atomic active job duplicate prevention under concurrent manual/scheduled attempts with slightly different `now` values, and DST edge cases.
- [x] 2.4 Implement scheduler integration that checks enabled users discovered from materialized `_superusers` settings, performs stale active-job recovery, runs/wakes a durable pending-job worker with transactional single-consumer claims and heartbeat updates, and starts due digest jobs with deterministic transactional/unique active-job claiming.

## 3. AI Digest Generation

- [x] 3.1 Add failing tests for Daily News prompt construction using entry summaries, takeaways, stars, sources, dates, IDs, 2000-code-point bounded/delimited user extra instructions, prompt-injection text in article fields, deterministic candidate capping, and candidate_count/included_count metadata.
- [x] 3.2 Implement AI digest generator that requests structured JSON containing title, Markdown body, and referenced entry IDs.
- [x] 3.3 Add failing tests for invalid AI references, duplicate reference deduplication, unvalidated inline `[[kh-entry:<entry_id>]]` markers, and malformed AI responses.
- [x] 3.4 Implement AI response parsing, same-user entry-reference validation, and safe failed-state recording.
- [x] 3.5 Add tests and implementation for empty windows producing a successful "No articles today" digest.

## 4. Manual Generation APIs

- [x] 4.1 Add failing route/API tests for authenticated asynchronous manual Generate now behavior, `202 Accepted` newly queued persisted pending jobs, worker pickup after route/process interruption, `200 OK` same-day successful digest idempotency, active job reuse, failed digest retry, owner scoping, and unauthenticated denial without job creation or existence leaks.
- [x] 4.2 Implement manual Generate now endpoint using authenticated-user-derived ownership, not generic digest collection mutation.
- [x] 4.3 Add failing route/API tests for Regenerate replacing an owned existing terminal digest only after success, preserving its period/local date, using explicit successful-snapshot/attempt-state fields, preserving prior successful content during active regeneration and after failed regeneration with sanitized error state, preserving the successful scheduled-day reservation after failed regeneration of a previously successful scheduled digest, returning existing active state without overwrite for pending/running selected digests or same-day/window active jobs, denying cross-user regeneration, and unauthenticated denial without mutation or existence leaks.
- [x] 4.4 Implement regeneration replacement behavior in a server-side route with status, content, references, counts, generated timestamp updates, and prior-success preservation on active/failed regeneration.
- [x] 4.5 Add concurrency tests proving the concrete database uniqueness/lock fields and indexes prevent duplicate active jobs for the same user/local date and canonical digest period, including scheduled/manual races with slightly different observed `now` values and pre-due manual versus later scheduled attempts.

## 5. Daily News Frontend

- [x] 5.1 Add failing UI/unit tests for Daily News navigation visibility and page loading states.
- [x] 5.2 Add Daily News navigation item and route.
- [x] 5.3 Implement latest digest display using route DTO raw `body_markdown` rendered only through the Daily News sanitizer component, explicit element/link allowlist, strict handling of raw HTML/images/dangerous URL schemes/untrusted links, subset indication, and newspaper-like visual styling.
- [ ] 5.4 Implement pending, failed, and "No articles today" UI states.
- [ ] 5.5 Add route-backed paginated or load-more previous digest browsing and selection with owner enforcement.
- [ ] 5.6 Add Generate now and Regenerate controls with loading and error states.

## 6. Entry Reference Modal

- [ ] 6.1 Add failing UI and route tests for opening an entry card from a Daily News reference through a digest-scoped endpoint, including digest ownership, referenced-entry membership, current entry visibility, sanitized DTO shape, unavailable state, and no cross-user existence leak.
- [ ] 6.2 Implement internal entry reference rendering from validated structured digest references and inline `[[kh-entry:<entry_id>]]` marker locations, not model-generated Markdown URLs.
- [ ] 6.3 Implement the digest-scoped entry-reference read route and entry-card modal behavior that reuses existing entry card display/actions where practical.
- [ ] 6.4 Add unavailable-entry handling when a referenced entry no longer exists or is not visible, while keeping archived digest body snapshots visible to the digest owner.

## 7. Daily News Settings UI

- [ ] 7.1 Add failing UI/API tests for reading and saving per-user Daily News settings through explicit GET/PUT route contracts, default materialization, `400` validation errors preserving previous values, unauthenticated settings denial, and extra-instruction length/character validation allowing printable Unicode plus `\t`, `\n`, and `\r` while rejecting other control/format characters.
- [ ] 7.2 Add settings controls for enablement, generation time, timezone, and extra digest instructions.
- [ ] 7.3 Validate IANA timezone values, local time format, and 2000-code-point extra-instruction limits in backend and frontend paths, preserving previous valid values on rejected saves.
- [ ] 7.4 Ensure saved extra instructions affect subsequent manual and scheduled generation.

## 8. Verification and Coverage

- [ ] 8.1 Run backend tests with coverage for Daily News scheduler, generator, routes, and collection logic.
- [ ] 8.2 Run frontend tests for Daily News page, settings, archive browsing, and modal interactions.
- [ ] 8.3 Run full project test suite and fix regressions.
- [ ] 8.4 Build the frontend and backend successfully.
- [ ] 8.5 Manually proof the feature in a tmux-run app session with generated sample data.
