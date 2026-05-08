## Why

KnowledgeHub currently summarizes individual articles, but it does not provide a concise daily briefing that helps a user understand the most important developments across all articles they received. A user-specific Daily News digest gives the application a higher-level "knowledge radar" view: what changed, what is breaking, and what may still be worth scanning.

## What Changes

- Add a Daily News option to the application navigation.
- Generate a user-specific daily digest from articles published or ingested since the user's last successful digest, or from the past 24 hours when no previous digest exists.
- Run digest generation daily at each user's configured local time, defaulting to 08:00 in Europe/Amsterdam.
- Add user-specific Daily News settings for enablement, generation time, timezone, extra editorial instructions, route-enforced owner scoping with database invariants, and an enforceable one-settings-record-per-user invariant. PocketBase `_superusers` remain fully privileged administrative identities; Daily News end-user behavior must use authenticated server-side routes that enforce ownership and must not rely on generic collection API rules as the isolation boundary for superuser tokens.
- Generate a newspaper-like structured Markdown digest using existing entry titles, sources, summaries, takeaways, dates, and effective star ratings.
- Organize the digest with the most important items first, using stars, recency, source context, AI-detected significance, breaking/developing signals, and the user's extra instructions.
- Include a dedicated breaking/developing section when relevant.
- Include a concise "You May Also Find This Interesting" section for lower-rated but potentially useful articles when relevant.
- Link referenced articles to KnowledgeHub entry cards through a route-level reference-read endpoint that verifies digest ownership, validates the entry is part of the digest references, re-checks current entry visibility, and returns sanitized entry-card DTO data or an unavailable state without leaking cross-user existence.
- Allow manual generation and regeneration through authenticated server-side routes with atomic duplicate-active-job handling based on canonical per-user/local-date/window keys; near-simultaneous scheduled and manual attempts for the same due window collapse to one active job, while pre-due ad-hoc manual digests do not suppress the later scheduled digest. Regeneration is the explicit exception to archive immutability: it targets the selected digest period, never overwrites while a same-day/window digest job is pending or running, preserves existing successful content while regeneration is active, replaces content only after success, and preserves prior successful content plus a sanitized failure state if regeneration fails.
- Retain previous digests indefinitely as immutable owner-visible snapshots and provide a paginated way to browse them.
- Create an explicit "No articles today" digest when there are no candidate entries.
- Surface pending or failed digest states when generation cannot complete, such as missing AI configuration or LLM failure, using sanitized user-safe error messages.
- Bound digest prompt size deterministically and record when only a subset of candidates was sent to the LLM.
- Use asynchronous manual generation routes that persist pending jobs before returning, a durable worker/claimer for `pending -> running -> success|failed` processing with heartbeat fields, stale active-job recovery after crashes/redeploys, and atomic active-job uniqueness.
- Define a digest DTO boundary where server routes return stored raw Markdown plus structured validated references to the first-party UI only; the frontend Daily News sanitizer component is the sole renderer and must apply the strict Markdown/link allowlist before display.
- Render digest Markdown through a strict sanitizer with an explicit Markdown/link allowlist, render KnowledgeHub entry references only from validated structured IDs and `[[kh-entry:<entry_id>]]` inline markers, and construct prompts so article/user text is treated as untrusted data rather than instructions.

## Capabilities

### New Capabilities
- `daily-news`: User-specific scheduled and manual Daily News digest generation, storage, browsing, rendering, settings, and KnowledgeHub entry references.

### Modified Capabilities
- `feed-view`: Add an in-app entry-card modal/deep-link behavior so Daily News references can open the specific KnowledgeHub article card.

## Impact

- Backend collections: new user-owned Daily News digest and settings storage, including digest trigger, deterministic lock-key fields backed by concrete unique indexes, successful-snapshot, attempt-state, and heartbeat fields.
- Backend scheduler: new per-user daily scheduling logic based on local time and timezone, including same-day missed-run catch-up without previous-day backfill.
- AI processing: new digest-generation prompt and parser using existing article summaries rather than raw article content.
- Routes/API: authenticated endpoints for settings, manual generation/regeneration, digest retrieval with explicit raw-Markdown DTO semantics, paginated digest retrieval, and digest-scoped entry-reference card reads, with unauthenticated requests denied before lookup or mutation.
- Frontend navigation and pages: Daily News page, archive browsing, settings controls, Markdown rendering, and entry-card modal behavior.
- Tests: scheduler timing, digest window selection, AI prompt behavior, settings persistence, failure states, archive pagination, and UI logic.
