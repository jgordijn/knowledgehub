## 1. Data Model and Test Fixtures

- [ ] 1.1 Add failing tests for `daily_news_settings` and `daily_digests` collection creation, owner-scoped auth rules, read-only user-facing digest collection access, persisted defaults, one-settings-record-per-user uniqueness, and user ownership.
- [ ] 1.2 Implement PocketBase collections for Daily News settings and digests, including a unique settings user index and denying generic user-facing digest create/update/delete rules.
- [ ] 1.3 Add testutil helpers for creating Daily News settings and digest records.
- [ ] 1.4 Add migration/backfill behavior or startup defaults by enumerating PocketBase `_superusers`, including users created after startup, with idempotent get-or-create/upsert behavior.

## 2. Digest Window and Scheduling Logic

- [ ] 2.1 Add failing tests for digest input window selection: previous successful digest, failed digest non-advancement, first 24-hour fallback, published_at match, and discovered_at match.
- [ ] 2.2 Implement digest candidate query logic using entries visible to the target user.
- [ ] 2.3 Add failing tests for timezone due checks, invalid timezone/time rejection, disabled settings, duplicate same-local-day prevention, atomic active job duplicate prevention under concurrent manual/scheduled attempts, and DST edge cases.
- [ ] 2.4 Implement scheduler integration that checks enabled users discovered from materialized `_superusers` settings and starts due digest jobs with transactional/unique active-job claiming.

## 3. AI Digest Generation

- [ ] 3.1 Add failing tests for Daily News prompt construction using entry summaries, takeaways, stars, sources, dates, IDs, bounded/delimited user extra instructions, prompt-injection text in article fields, deterministic candidate capping, and candidate_count/included_count metadata.
- [ ] 3.2 Implement AI digest generator that requests structured JSON containing title, Markdown body, and referenced entry IDs.
- [ ] 3.3 Add failing tests for invalid AI references and malformed AI responses.
- [ ] 3.4 Implement AI response parsing, same-user entry-reference validation, and safe failed-state recording.
- [ ] 3.5 Add tests and implementation for empty windows producing a successful "No articles today" digest.

## 4. Manual Generation APIs

- [ ] 4.1 Add failing route/API tests for authenticated manual Generate now behavior, same-day successful digest idempotency, active job reuse, failed digest retry, and owner scoping.
- [ ] 4.2 Implement manual Generate now endpoint using authenticated-user-derived ownership, not generic digest collection mutation.
- [ ] 4.3 Add failing route/API tests for Regenerate overwriting an owned existing digest, preserving its period/local date, and denying cross-user regeneration.
- [ ] 4.4 Implement regeneration overwrite behavior in a server-side route with status, content, references, counts, and generated timestamp updates.
- [ ] 4.5 Add concurrency tests proving the database uniqueness/lock prevents duplicate active jobs for the same user and digest period.

## 5. Daily News Frontend

- [ ] 5.1 Add failing UI/unit tests for Daily News navigation visibility and page loading states.
- [ ] 5.2 Add Daily News navigation item and route.
- [ ] 5.3 Implement latest digest display with sanitized Markdown rendering, strict handling of raw HTML/images/untrusted links, subset indication, and newspaper-like visual styling.
- [ ] 5.4 Implement pending, failed, and "No articles today" UI states.
- [ ] 5.5 Add paginated or load-more previous digest browsing and selection.
- [ ] 5.6 Add Generate now and Regenerate controls with loading and error states.

## 6. Entry Reference Modal

- [ ] 6.1 Add failing UI tests for opening an entry card from a Daily News reference.
- [ ] 6.2 Implement internal entry reference rendering from validated structured digest references, not model-generated Markdown URLs.
- [ ] 6.3 Implement entry-card modal behavior that reuses existing entry card display/actions where practical.
- [ ] 6.4 Add unavailable-entry handling when a referenced entry no longer exists or is not visible.

## 7. Daily News Settings UI

- [ ] 7.1 Add failing UI/API tests for reading and saving per-user Daily News settings.
- [ ] 7.2 Add settings controls for enablement, generation time, timezone, and extra digest instructions.
- [ ] 7.3 Validate IANA timezone values and local time format in backend and frontend paths, preserving previous valid values on rejected saves.
- [ ] 7.4 Ensure saved extra instructions affect subsequent manual and scheduled generation.

## 8. Verification and Coverage

- [ ] 8.1 Run backend tests with coverage for Daily News scheduler, generator, routes, and collection logic.
- [ ] 8.2 Run frontend tests for Daily News page, settings, archive browsing, and modal interactions.
- [ ] 8.3 Run full project test suite and fix regressions.
- [ ] 8.4 Build the frontend and backend successfully.
- [ ] 8.5 Manually proof the feature in a tmux-run app session with generated sample data.
