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

- `daily_news_settings`: one record per user with enabled flag, generation time, timezone, and extra prompt.
- `daily_digests`: user-owned generated digest records with period, status, body, referenced entries, counts, and error state.

Rationale: the settings are user-specific and include scheduling behavior, while digests need history and status. A dedicated schema is clearer than key/value settings.

Alternative considered: add more keys to `app_settings`. Rejected because `app_settings` is currently global and key/value storage would make per-user scheduling and validation harder.

### Use a per-user scheduler loop with local-time due checks

Keep one backend scheduler loop that periodically checks all enabled `daily_news_settings` records. For each user, convert the current instant into the configured timezone and determine whether the configured local time is due and not already generated for that local date.

Rationale: one loop is simpler than maintaining many individual timers and handles timezone changes, restarts, and missed runs consistently.

Alternative considered: schedule one timer per user. Rejected for more lifecycle complexity and less robust restart behavior.

### Select entries since last successful digest, falling back to 24 hours

For automatic generation, set `period_start` to the previous successful digest's `period_end` for that user. If none exists, use `now - 24h`. Set `period_end` to the generation time. Include entries visible to that user whose `published_at` or `discovered_at` falls inside the window.

Rationale: this avoids gaps after delayed runs while still supporting the first run.

Trade-off: an article with old publication date but newly ingested in the window can still be included through `discovered_at`, matching the requested "published or ingested" behavior.

### Generate from existing summaries and metadata

The digest prompt uses entry title, source, published/discovered times, effective stars, summary, takeaways, and entry ID. Raw article content is not included by default.

Rationale: this controls token cost and makes digest quality depend on the already-tested article summarization pipeline.

Alternative considered: include raw content for top entries. Deferred until there is evidence summaries are insufficient.

### Require structured AI output plus Markdown body

Ask the LLM for JSON containing at least:

- title
- body_markdown
- referenced_entry_ids
- optional breaking_entry_ids
- optional interesting_entry_ids

The Markdown body is rendered for readability. The structured IDs let the UI render safe in-app entry links/modals without trusting arbitrary Markdown URLs from the model.

Rationale: Markdown gives a good writing format; structured references keep linking deterministic and testable.

### Render article references as in-app entry card modals

Daily News references should open the KnowledgeHub entry inside the app, initially as an entry-card modal. The modal can reuse existing entry card display logic and actions where practical.

Rationale: the user asked to inspect the specific KnowledgeHub card, not jump directly to the original article. A modal keeps the reader in the digest context.

Alternative considered: add a full `/entries/:id` detail route. This can be added later, but a modal is smaller for the first version.

### Regeneration overwrites the selected digest

Manual regeneration for an existing digest updates that digest's content, referenced entries, status, counts, and generated timestamp. It does not create a revision history.

Rationale: this keeps first-version data model and UI simple.

### Keep digests indefinitely with paginated browsing

Do not prune digests in the first version. The Daily News page shows the latest digest prominently and previous editions through pagination or "Load more".

Rationale: daily Markdown records are small and history is useful.

## Risks / Trade-offs

- **LLM output references nonexistent or omitted entries** → Validate returned entry IDs against the candidate set before storing/rendering links.
- **Digest generation could exceed token limits for high-volume days** → Cap or batch candidate input by importance signals; preserve count metadata and ask the model to mention if not all items were included. Prefer summaries over raw content.
- **Timezone scheduling bugs around daylight saving time** → Store IANA timezone names and compare local dates/times using Go timezone APIs in tests covering DST boundaries.
- **Existing entries may not be fully user-owned** → Store Daily News records per user and query entries according to current app visibility. If entries later become user-owned, the digest query can be narrowed without changing the digest contract.
- **Manual regeneration during automatic generation can race** → Use digest status and per-user/date lookup to avoid duplicate active jobs. Last successful overwrite wins for explicit regeneration.
- **Markdown rendering security** → Render sanitized Markdown and intercept internal entry references through structured IDs rather than arbitrary model-generated HTML.
- **Missing OpenRouter configuration** → Store a failed digest state with a clear error rather than silently skipping, so the page can explain why no digest was generated.
