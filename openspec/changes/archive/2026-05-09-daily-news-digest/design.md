## Context

KnowledgeHub already creates article-level summaries and star ratings via OpenRouter and stores entries in PocketBase. The requested Daily News feature adds a higher-level user-specific briefing that summarizes a time window of entries into a newspaper-like digest. The digest must be generated on a schedule, configurable per user, and viewable later from a new navigation item.

The existing app has global `app_settings` for AI configuration. Daily News settings are different: generation time, timezone, enablement, and editorial instructions must be user-specific. Digest output must also belong to a user so multiple users can have independent schedules, prompts, and archives.

Digest generation should use existing entry summaries/takeaways and metadata rather than raw article content. This keeps cost and token usage predictable and reuses the article-level AI work already done by the ingestion pipeline.

## Goals / Non-Goals

**Goals:**
- Provide each user with a scheduled daily briefing of recent KnowledgeHub entries.
- Allow users to configure daily generation time, timezone, enablement, and extra editorial prompt instructions.
- Store digests historically and display the latest digest plus paginated prior editions.
- Render a structured Markdown digest with newspaper-like sections, including breaking/developing items and lower-rated interesting items.
- Link referenced digest articles to in-app KnowledgeHub entry cards.
- Allow manual generation and regeneration, with regeneration as the only explicit exception to digest immutability: it preserves the selected digest period and replaces stored content only after successful regeneration.
- Make empty, pending, and failed states explicit and testable.

**Non-Goals:**
- Email delivery or push notifications for digest completion.
- External news discovery outside configured KnowledgeHub resources.
- Full topic-clustering infrastructure beyond what the digest prompt can infer from the selected entries.
- Revision history for regenerated digests.
- A full multi-user resource isolation rewrite if existing entries are not yet user-owned; this change stores digest ownership and uses the existing entry visibility model.

## Decisions

### Store Daily News in dedicated collections

Create dedicated collections rather than overloading `app_settings`:

- `daily_news_settings`: one record per user with `user`, enabled flag, generation time, timezone, and extra prompt capped at 2000 Unicode code points.
- `daily_digests`: user-owned generated digest records with `user`, local digest date, period, status, trigger (`automatic` or `manual`), canonical active keys, body, referenced entries, candidate/included counts, subset indicator, successful-snapshot metadata, attempt timestamps/heartbeat, and sanitized error state.

`daily_news_settings` must enforce one record per user with a database-level unique index on `user`; settings creation/update must be idempotent get-or-create/upsert behavior so duplicate settings cannot create ambiguous scheduler state. User-facing generic collection access is owner-scoped and read-only by default as defense in depth: list/view may only return records whose `user` matches `@request.auth.id`, while generic create/delete are denied. Because PocketBase `_superusers` are administrative identities that can bypass collection rules, those generic rules are not the Daily News isolation boundary for superuser tokens. The frontend and supported end-user API must use server-side get-or-create/update routes that derive the user from the authenticated request, enforce ownership in route code before lookup or mutation, and never accept an arbitrary owner ID. If direct generic update is enabled for non-superuser auth in the future, it must still be owner-scoped and must preserve the `user` field and uniqueness invariant. Unauthenticated settings requests are denied without materializing anonymous settings. Extra prompt instructions are validated in both backend and frontend paths with a 2000-code-point maximum; oversized values are rejected and previous valid settings remain unchanged. Supported characters are printable Unicode scalar values plus horizontal tab, line feed, and carriage return (`\t`, `\n`, `\r`); all other Unicode control/format characters (including other `Cc` and `Cf` code points) are rejected before storage by a shared validation helper used by backend validation and frontend-facing validation.

`daily_digests` must be read-only through user-facing collection rules as defense in depth: owner-scoped list/view are allowed, while create/update/delete are denied through the generic collection API. `_superusers` remain fully privileged administrators and may bypass those collection rules; therefore supported Daily News end-user reads and mutations must go through server-side routes that derive the user from the authenticated request and perform explicit owner checks before lookup, response, or mutation. All digest mutations, including manual generation, regeneration, status updates, failure recording, stale-job recovery, and any future delete action, must happen through server-side code/routes rather than accepting arbitrary user IDs. Server-side mutation code must validate structured entry references against entries visible to that same user before storage or rendering and must sanitize all user-visible failure fields. Digest read DTOs return the stored raw `body_markdown` and structured reference metadata only to the authenticated first-party UI; routes do not return trusted HTML. The frontend Daily News Markdown renderer is the sole raw-Markdown renderer and must apply the strict sanitizer/allowlist before inserting content into the DOM.

The digest record distinguishes the latest generation attempt from the last successful visible snapshot. `status` represents the current/latest attempt (`pending`, `running`, `success`, or `failed`). `body_markdown`, `title`, `referenced_entry_ids`, `candidate_count`, `included_count`, `used_subset`, and `last_success_at` represent the last successful snapshot and are not cleared when a regeneration is pending/running or fails. `has_successful_snapshot` explicitly tells the UI whether those snapshot fields are meaningful. `error_message` and `attempt_finished_at` describe the latest failed attempt only and must be sanitized. `queued_at`, `started_at`, and `heartbeat_at` support active-job processing and stale recovery. `window_key`, `active_window_key`, `scheduled_day_key`, `active_scheduled_day_key`, and `successful_scheduled_day_key` are deterministic server-derived lock fields used by SQLite unique indexes for atomic concurrency protection. A first-time failed digest has `has_successful_snapshot=false`; a failed regeneration of a previously successful digest has `status=failed`, `has_successful_snapshot=true`, preserved snapshot fields, and a latest-attempt error.

Rationale: the settings are user-specific and include scheduling behavior, while digests need history and status. A dedicated schema is clearer than key/value settings.

Alternative considered: add more keys to `app_settings`. Rejected because `app_settings` is currently global and key/value storage would make per-user scheduling and validation harder.

### Materialize default settings for users

Daily News defaults are persisted in `daily_news_settings`. KnowledgeHub currently authenticates the app through PocketBase `_superusers`, so `_superusers` is the source of truth for Daily News owner IDs until a dedicated non-superuser auth collection exists. `_superusers` are treated as trusted administrative credentials, not as mutually isolated untrusted tenants; collection rules are not relied on to constrain a malicious superuser. The supported Daily News UI/API path is server-side route access with explicit owner enforcement. On startup and during each Daily News scheduler/settings flow, the system enumerates `_superusers` and ensures each authenticated owner has exactly one settings record using enabled=true, generation time 08:00, and timezone Europe/Amsterdam unless that owner has already saved settings. This lets the scheduler cover owners who have not opened the settings page and owners created after startup.

### Use a per-user scheduler loop with local-time due checks

Keep one backend scheduler loop that periodically checks all enabled `daily_news_settings` records, including materialized default records. For each user, convert the current instant into the configured timezone and determine whether the configured local time is due and not already generated or active for that local date. A digest is due when the current local date equals the target local date and the current local clock is at or after the configured generation time. This intentionally catches same-day missed runs after downtime. The scheduler does not automatically backfill previous local dates once the user's local date has advanced; it evaluates the current local date only.

Rationale: one loop is simpler than maintaining many individual timers and handles timezone changes, restarts, and missed runs consistently.

Alternative considered: schedule one timer per user. Rejected for more lifecycle complexity and less robust restart behavior.

### Use deterministic SQLite-backed digest lock keys

Daily News uses concrete server-derived lock-key fields instead of application-level prechecks. `window_key` is always `user|local_date|period_start_utc|period_end_utc` using canonical UTC timestamps at fixed precision. For active records, `active_window_key=window_key`; for terminal records, `active_window_key` is empty or NULL. Scheduled/due jobs additionally set `scheduled_day_key=user|local_date`, `active_scheduled_day_key=scheduled_day_key` while pending/running, and `successful_scheduled_day_key=scheduled_day_key` only after successful automatic/scheduled completion. Ad-hoc pre-due manual jobs leave scheduled-day keys empty but still use `window_key`/`active_window_key`. PocketBase startup/migration code creates SQLite unique indexes over non-empty `active_window_key`, non-empty `active_scheduled_day_key`, and non-empty `successful_scheduled_day_key` (or equivalent unique nullable fields where SQLite allows multiple NULL values). The claim routine computes and writes these keys inside the same transaction that inserts or updates the digest record. Terminal transitions clear active keys. Success transitions set the appropriate success key. First-time failed jobs do not set success keys. Failed regeneration of a digest that already has a successful scheduled snapshot preserves its existing `successful_scheduled_day_key` (or equivalent immutable success-reservation field) while preserving the successful snapshot, so the one-successful-scheduled-digest invariant remains reserved until a later successful regeneration replaces the content in place. Tests must prove concurrent scheduled/manual attempts cannot bypass these indexes with slightly different observed `now` instants.

### Recover stale active jobs after crashes

On startup and before each scheduler claim pass, Daily News performs stale-job recovery for `pending` and `running` digests. A `pending` job that has not been picked up within the configured pending timeout is either resumed by claiming it for processing or marked `failed` with a sanitized timeout message before a new claim is allowed. A `running` job whose `started_at`/heartbeat is older than the configured running timeout is considered abandoned after process crash or redeploy and is marked `failed` with a sanitized timeout message. Recovery must not mark a job stale solely because LLM generation is slow within the timeout, and timeout values must be deterministic/testable. Failed stale jobs do not advance the automatic input window and do not reserve active keys, so scheduled or manual retry can create a new pending job for the canonical window.

Rationale: without recovery, a stale active record could permanently block scheduled generation or regeneration for that local day/window.

### Select entries since last successful digest, falling back to 24 hours

For automatic generation, set `period_start` to the previous successful digest's canonical `period_end` for that user. If none exists, use `period_end - 24h`. Set `period_end` to the canonical generation instant for the user's local digest date: the configured `HH:MM` interpreted in the configured timezone for scheduled/due same-day generation, stored in UTC with a deterministic precision. Manual Generate now uses the same canonical due instant when the user's configured generation time is already due for that local date. Before the configured time is due, Generate now creates an ad-hoc manual digest with a deterministic manual `period_end` normalized by the claim routine and `trigger=manual`; it does not count as that day's successful scheduled digest and therefore must not suppress the later automatic run. The later automatic run still uses its canonical due instant, and its input window starts after the latest successful digest period end for the user, including any earlier same-day manual digest, so entries between the manual digest and the scheduled due time are still covered. The claim routine derives `local_date`, `period_start`, `period_end`, `trigger`, and `job_key` together inside the same transactional path. Duplicate active-job protection uses canonical `(user, local_date, period_start, period_end)` window keys and, for scheduled/due generation, an active scheduled-day guard; tiny differences between scheduled and manual `now` values cannot bypass duplicate-active-job protection. Include entries visible to that user whose `published_at` or `discovered_at` falls inside the canonical window. Failed digests do not advance the next automatic window; only successful digests provide the previous `period_end`.

Rationale: this avoids gaps after delayed runs while still supporting the first run.

Trade-off: an article with old publication date but newly ingested in the window can still be included through `discovered_at`, matching the requested "published or ingested" behavior.

### Generate from existing summaries and metadata

The digest prompt uses entry title, source, published/discovered times, effective stars, summary, takeaways, and entry ID. Raw article content is not included by default.

Rationale: this controls token cost and makes digest quality depend on the already-tested article summarization pipeline.

Alternative considered: include raw content for top entries. Deferred until there is evidence summaries are insufficient.

### Require structured AI output plus Markdown body

Treat every article field and user extra instruction as untrusted data when constructing the prompt. Entry titles, summaries, takeaways, source names, and user instructions must be wrapped in explicit delimiters or encoded sections, and the system prompt must instruct the model not to follow instructions contained inside those data fields. User extra instructions may influence editorial priorities only within the Daily News task and must be bounded/sanitized before inclusion.

Ask the LLM for JSON containing at least:

- title
- body_markdown
- referenced_entry_ids
- optional breaking_entry_ids
- optional interesting_entry_ids

The Markdown body is rendered for readability. When the model wants an inline KnowledgeHub control, it must place a plain marker in `body_markdown` using exactly `[[kh-entry:<entry_id>]]` and include the same ID in `referenced_entry_ids`. During parsing, returned IDs are validated against the candidate set and current user visibility, deduplicated for storage by first appearance, and never trusted merely because they appear in Markdown. During rendering, only markers whose IDs are present in the validated stored references become in-app entry controls at the marker location; invalid, duplicate-only, or unreferenced markers are rendered as inert text or removed according to the sanitizer. The structured IDs let the UI render safe in-app entry links/modals without trusting arbitrary Markdown URLs from the model.

Rationale: Markdown gives a good writing format; structured references keep linking deterministic and testable.

### Treat digest Markdown as an immutable owned snapshot

A stored digest `body_markdown` is an immutable historical snapshot of the digest that was generated for its owner at that time, except when the owner explicitly invokes Regenerate for that digest. It may contain copied titles, summaries, source names, and takeaways from entries that were visible to that owner during generation. If a referenced entry is later deleted or becomes no longer visible to that same owner, the archived digest body remains visible to the digest owner, but structured entry links and entry-card controls for the unavailable entry must be removed or shown as unavailable. Cross-user access remains denied by server-side owner checks and defense-in-depth collection rules. This policy avoids silently rewriting retained archives while still preventing stale structured references from opening inaccessible entries.

### Render article references as in-app entry card modals

Daily News references should open the KnowledgeHub entry inside the app, initially as an entry-card modal. The modal can reuse existing entry card display logic and actions where practical. The frontend must fetch reference-card data through a digest-scoped server route such as `GET /api/daily-news/digests/{digestId}/entries/{entryId}` rather than a generic entries collection read. The route derives the caller from authentication, verifies the digest belongs to the caller, verifies `entryId` is present in the digest's validated `referenced_entry_ids`, re-checks current entry visibility for that caller, and returns a sanitized entry-card DTO or an unavailable response/state. Cross-user or non-referenced IDs must use an auth-safe not-found/denied response that does not reveal whether the entry exists.

Rationale: the user asked to inspect the specific KnowledgeHub card, not jump directly to the original article. A modal keeps the reader in the digest context.

Alternative considered: add a full `/entries/:id` detail route. This can be added later, but a modal is smaller for the first version.

### Manual generation and regeneration are asynchronous and idempotent per user/day

Manual "Generate now" derives the current window for the authenticated user using the same canonical window claim routine as the scheduler. If the configured daily time is already due, the manual request targets the canonical scheduled window and reuses an active or successful scheduled digest for that local date. If the configured daily time is not yet due, the manual request targets an ad-hoc manual window and may complete without counting as the day's scheduled digest; the scheduled run remains due later for the same local date. If a pending or running digest already exists for the same user/local-date/window, the route returns that active digest instead of creating a second job. Duplicate active-job prevention must be atomic: generation creates or claims a deterministic per-user/local-date/window `job_key` and, for scheduled/due generation, an active scheduled-day guard (or equivalent lock keys) inside a transaction, backed by database uniqueness constraints or equivalent locks, so concurrent manual and scheduled attempts cannot both insert equivalent active jobs even if their observed `now` values differ by milliseconds. A digest's active lifecycle is `pending -> running -> success|failed`; only `pending` and `running` records count as active jobs. Failed digests remain historical failure records and do not reserve the active key for retry. If a successful scheduled digest already exists for the same user and local digest date, Generate now returns the existing digest and asks the user to use Regenerate for an explicit overwrite. Failed digests do not block a new Generate now request and do not advance the next automatic window.

Automatic digest uniqueness is scoped to at most one successful scheduled digest per `(user, local_date)`. Active-job uniqueness is scoped to at most one `pending` or `running` digest per canonical `(user, local_date, period_start, period_end)` window, plus at most one active scheduled/due digest per `(user, local_date)`, using deterministic `active_scheduled_day_key` and `job_key`/`window_key` values. Pre-due ad-hoc manual digests do not use the scheduled-day success key, but they still use canonical window keys. The key input uses UTC-normalized timestamps with a fixed precision and is produced only by the shared claim routine. Regeneration is explicit: it updates the selected digest record in place rather than inserting a second successful digest for the same period, preserving history for other days while avoiding same-day automatic duplicates.

Manual generation routes are asynchronous. A newly claimed job returns `202 Accepted` with the digest/job record identifier and initial `pending` state; active-job reuse also returns the existing active record; existing successful digests return `200 OK` with that digest. Route handlers persist the pending job before returning and then signal the Daily News job runner; correctness must not depend on an in-request goroutine starting successfully. The Daily News page observes completion by polling or PocketBase realtime updates on the returned digest record.

Daily News processing uses a durable in-process worker loop plus a single-consumer claim routine. On startup and scheduler ticks, the worker scans for pending jobs after stale-job recovery; route handlers also wake/signal the worker after inserting pending jobs. Claiming a job is transactional: update exactly one `pending` record to `running`, set `started_at` and `heartbeat_at`, and proceed only if the compare-and-set succeeds. Running jobs update `heartbeat_at` at deterministic intervals during long work. If the process exits after a route returns `202 Accepted` but before any goroutine starts, the persisted pending job is picked up by startup/scheduler scanning or marked failed by pending-timeout recovery according to the stale-job policy.

Manual regeneration for an existing digest is allowed only for the owner and only when neither the selected digest nor another digest for the same user/local-date/window is `pending` or `running`. If the selected digest is active, or if a same-day/window active job exists, the route returns or displays that active digest state and does not overwrite content or create a second job. For a previously successful digest, regeneration marks the record as active but preserves the previous successful `body_markdown`, title, references, and counts for display until replacement content has been generated successfully. On regeneration success, the selected digest's content, references, status, counts, and generated timestamp are replaced while preserving the selected digest's original `period_start`, `period_end`, and local digest date. On regeneration failure, any previous successful body/references remain visible as the last successful snapshot, and the record stores a sanitized failure state/message so the UI can report that the attempted regeneration failed without losing the prior digest. For a previously failed digest with no successful body, a failed regeneration stores only the new sanitized failure state. It does not create a revision history. Unauthenticated manual generation and regeneration requests are denied before lookup or mutation so they do not create jobs or reveal digest existence.

Rationale: this keeps first-version data model and UI simple.

### Expose authenticated Daily News server routes

Supported Daily News UI access uses explicit server routes rather than generic collection APIs. `GET /api/daily-news/settings` requires authentication, materializes the caller's default settings when missing, and returns `200 OK` with the caller-owned settings DTO. `PUT /api/daily-news/settings` requires authentication, derives `user` from the request context, ignores/rejects any owner field in the body, validates enabled/timezone/generation time/extra instructions, returns `200 OK` with the saved settings on success, and returns `400 Bad Request` with a sanitized validation error while preserving previous values on invalid input. Unauthenticated settings requests return `401 Unauthorized` or the app's standard auth-denied status before lookup/materialization.

Digest reads use route-level owner enforcement as well: `GET /api/daily-news/digests` returns the caller's latest digest and paginated archive metadata, and `GET /api/daily-news/digests/{id}` returns a caller-owned digest DTO or an auth-safe not-found/denied response without revealing other users' digest contents. Digest DTOs include raw `body_markdown`, snapshot/attempt metadata, and validated structured references, but no pre-trusted HTML; the first-party UI sanitizer is responsible for rendering. `GET /api/daily-news/digests/{digestId}/entries/{entryId}` returns sanitized entry-card DTO data only when the caller owns the digest, the entry is a validated digest reference, and the entry remains visible to the caller; otherwise it returns an unavailable/auth-safe response. `POST /api/daily-news/generate` performs asynchronous Generate now and returns `202 Accepted` for a newly queued pending job or an active existing job, and `200 OK` for an existing successful scheduled digest. `POST /api/daily-news/digests/{id}/regenerate` queues or returns the selected owned digest according to the regeneration rules. All unauthenticated mutation/read routes are denied before lookup or mutation.

### Keep digests indefinitely with paginated browsing

Do not prune digests in the first version. The Daily News page shows the latest digest prominently and previous editions through pagination or "Load more".

Rationale: daily Markdown records are small and history is useful.

## Risks / Trade-offs

- **LLM output references nonexistent or omitted entries** → Validate returned entry IDs against the candidate set before storing/rendering links.
- **Digest generation could exceed token limits for high-volume days** → Build prompts from a deterministic preselection ordered by effective stars, recency, breaking/developing signals, and source/title tie-breakers. Store `candidate_count`, `included_count`, and a subset indicator; the UI must show when a digest was based on a subset. Prefer summaries over raw content.
- **Timezone scheduling bugs around daylight saving time** → Store IANA timezone names and compare local dates/times using Go timezone APIs in tests covering DST boundaries. Reject invalid timezone names and invalid `HH:MM` generation times on save.
- **Existing entries may not be fully user-owned** → Store Daily News records per user and query entries according to current app visibility. If entries later become user-owned, the digest query can be narrowed without changing the digest contract.
- **Manual regeneration during automatic generation can race** → Use digest status plus deterministic per-user/date and per-user/date/window locks to avoid duplicate active jobs. Regeneration is blocked or returns the active state while any same-day/window job is pending/running; after terminal state, an explicit regeneration may overwrite the selected digest, replacing prior content only after successful generation.
- **Markdown rendering security** → Render sanitized Markdown with an explicit allowlist: headings, paragraphs, emphasis/strong, blockquotes, ordered/unordered lists, tables, and inline/fenced code are allowed; raw HTML, scripts, event-handler attributes, iframes, styles, SVG, and images are removed. Markdown links are either rendered as plain text or allowed only for `https://` URLs with `rel="noopener noreferrer"` and safe targets; `javascript:`, `data:`, `file:`, protocol-relative, and other schemes are removed or neutralized. Intercept internal entry references through structured IDs rather than arbitrary model-generated HTML or trusted Markdown URLs.
- **Missing OpenRouter configuration** → Store a failed digest state with a clear, user-safe error rather than silently skipping, so the page can explain why no digest was generated. Do not store or display API keys, provider payloads, stack traces, or other secrets in digest error fields.
