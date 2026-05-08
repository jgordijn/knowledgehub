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
- Allow manual generation and regeneration, with regeneration overwriting the selected digest's content for the first version.
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

- `daily_news_settings`: one record per user with `user`, enabled flag, generation time, timezone, and extra prompt.
- `daily_digests`: user-owned generated digest records with `user`, local digest date, period, status, body, referenced entries, candidate/included counts, subset indicator, and sanitized error state.

`daily_news_settings` must enforce one record per user with a database-level unique index on `user`; settings creation/update must be idempotent get-or-create/upsert behavior so duplicate settings cannot create ambiguous scheduler state. User-facing generic collection access is owner-scoped and read-only by default: list/view may only return records whose `user` matches `@request.auth.id`, while generic create/delete are denied. User edits happen through server-side get-or-create/update routes that derive the user from the authenticated request and never accept an arbitrary owner ID. If direct generic update is enabled, it must still be owner-scoped and must preserve the `user` field and uniqueness invariant.

`daily_digests` must be read-only through user-facing collection rules: owner-scoped list/view are allowed, while create/update/delete are denied through the generic collection API. All digest mutations, including manual generation, regeneration, status updates, failure recording, and any future delete action, must happen through server-side code/routes that derive the user from the authenticated request rather than accepting arbitrary user IDs. Server-side mutation code must validate structured entry references against entries visible to that same user before storage or rendering and must sanitize all user-visible failure fields.

Rationale: the settings are user-specific and include scheduling behavior, while digests need history and status. A dedicated schema is clearer than key/value settings.

Alternative considered: add more keys to `app_settings`. Rejected because `app_settings` is currently global and key/value storage would make per-user scheduling and validation harder.

### Materialize default settings for users

Daily News defaults are persisted in `daily_news_settings`. KnowledgeHub currently authenticates users through PocketBase `_superusers`, so `_superusers` is the source of truth for Daily News owners until a dedicated auth collection exists. On startup and during each Daily News scheduler/settings flow, the system enumerates `_superusers` and ensures each authenticated user has exactly one settings record using enabled=true, generation time 08:00, and timezone Europe/Amsterdam unless that user has already saved settings. This lets the scheduler cover users who have not opened the settings page and users created after startup.

### Use a per-user scheduler loop with local-time due checks

Keep one backend scheduler loop that periodically checks all enabled `daily_news_settings` records, including materialized default records. For each user, convert the current instant into the configured timezone and determine whether the configured local time is due and not already generated for that local date.

Rationale: one loop is simpler than maintaining many individual timers and handles timezone changes, restarts, and missed runs consistently.

Alternative considered: schedule one timer per user. Rejected for more lifecycle complexity and less robust restart behavior.

### Select entries since last successful digest, falling back to 24 hours

For automatic generation, set `period_start` to the previous successful digest's `period_end` for that user. If none exists, use `now - 24h`. Set `period_end` to the generation time. Include entries visible to that user whose `published_at` or `discovered_at` falls inside the window. Failed digests do not advance the next automatic window; only successful digests provide the previous `period_end`.

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

The Markdown body is rendered for readability. The structured IDs let the UI render safe in-app entry links/modals without trusting arbitrary Markdown URLs from the model.

Rationale: Markdown gives a good writing format; structured references keep linking deterministic and testable.

### Treat digest Markdown as an immutable owned snapshot

A stored digest `body_markdown` is an immutable historical snapshot of the digest that was generated for its owner at that time. It may contain copied titles, summaries, source names, and takeaways from entries that were visible to that owner during generation. If a referenced entry is later deleted or becomes no longer visible to that same owner, the archived digest body remains visible to the digest owner, but structured entry links and entry-card controls for the unavailable entry must be removed or shown as unavailable. Cross-user access remains denied by digest ownership rules. This policy avoids silently rewriting retained archives while still preventing stale structured references from opening inaccessible entries.

### Render article references as in-app entry card modals

Daily News references should open the KnowledgeHub entry inside the app, initially as an entry-card modal. The modal can reuse existing entry card display logic and actions where practical.

Rationale: the user asked to inspect the specific KnowledgeHub card, not jump directly to the original article. A modal keeps the reader in the digest context.

Alternative considered: add a full `/entries/:id` detail route. This can be added later, but a modal is smaller for the first version.

### Manual generation and regeneration are asynchronous and idempotent per user/day

Manual "Generate now" derives the current window for the authenticated user. If a pending or running digest already exists for that same user and local digest date/window, the route returns that active digest instead of creating a second job. Duplicate active-job prevention must be atomic: generation creates or claims a deterministic per-user/local-date/window `job_key` (or equivalent lock key) inside a transaction, backed by a database uniqueness constraint or equivalent lock, so concurrent manual and scheduled attempts cannot both insert active jobs. A digest's active lifecycle is `pending -> running -> success|failed`; only `pending` and `running` records count as active jobs. Failed digests remain historical failure records and do not reserve the active key for retry. If a successful automatic digest already exists for the same user and local digest date, Generate now returns the existing digest and asks the user to use Regenerate for an explicit overwrite. Failed digests do not block a new Generate now request and do not advance the next automatic window.

Automatic digest uniqueness is scoped to at most one successful automatic digest per `(user, local_date)`. Active-job uniqueness is scoped to at most one `pending` or `running` digest per `(user, local_date, period_start, period_end)` using a deterministic `job_key`/`window_key`. Regeneration is explicit: it updates the selected digest record in place rather than inserting a second successful digest for the same period, preserving history for other days while avoiding same-day automatic duplicates.

Manual generation routes are asynchronous. A newly claimed job returns `202 Accepted` with the digest/job record identifier and initial `pending` state; active-job reuse also returns the existing active record; existing successful digests return `200 OK` with that digest. The Daily News page observes completion by polling or PocketBase realtime updates on the returned digest record.

Manual regeneration for an existing digest is allowed only for the owner. It updates that digest's content, referenced entries, status, counts, and generated timestamp while preserving the selected digest's original `period_start`, `period_end`, and local digest date. It does not create a revision history.

Rationale: this keeps first-version data model and UI simple.

### Keep digests indefinitely with paginated browsing

Do not prune digests in the first version. The Daily News page shows the latest digest prominently and previous editions through pagination or "Load more".

Rationale: daily Markdown records are small and history is useful.

## Risks / Trade-offs

- **LLM output references nonexistent or omitted entries** → Validate returned entry IDs against the candidate set before storing/rendering links.
- **Digest generation could exceed token limits for high-volume days** → Build prompts from a deterministic preselection ordered by effective stars, recency, breaking/developing signals, and source/title tie-breakers. Store `candidate_count`, `included_count`, and a subset indicator; the UI must show when a digest was based on a subset. Prefer summaries over raw content.
- **Timezone scheduling bugs around daylight saving time** → Store IANA timezone names and compare local dates/times using Go timezone APIs in tests covering DST boundaries. Reject invalid timezone names and invalid `HH:MM` generation times on save.
- **Existing entries may not be fully user-owned** → Store Daily News records per user and query entries according to current app visibility. If entries later become user-owned, the digest query can be narrowed without changing the digest contract.
- **Manual regeneration during automatic generation can race** → Use digest status and per-user/date lookup to avoid duplicate active jobs. Last successful overwrite wins for explicit regeneration.
- **Markdown rendering security** → Render sanitized Markdown with an explicit allowlist: headings, paragraphs, emphasis/strong, blockquotes, ordered/unordered lists, tables, and inline/fenced code are allowed; raw HTML, scripts, event-handler attributes, iframes, styles, SVG, and images are removed. Markdown links are either rendered as plain text or allowed only for `https://` URLs with `rel="noopener noreferrer"` and safe targets; `javascript:`, `data:`, `file:`, protocol-relative, and other schemes are removed or neutralized. Intercept internal entry references through structured IDs rather than arbitrary model-generated HTML or trusted Markdown URLs.
- **Missing OpenRouter configuration** → Store a failed digest state with a clear, user-safe error rather than silently skipping, so the page can explain why no digest was generated. Do not store or display API keys, provider payloads, stack traces, or other secrets in digest error fields.
