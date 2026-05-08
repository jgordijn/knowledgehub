## Why

KnowledgeHub currently summarizes individual articles, but it does not provide a concise daily briefing that helps a user understand the most important developments across all articles they received. A user-specific Daily News digest gives the application a higher-level "knowledge radar" view: what changed, what is breaking, and what may still be worth scanning.

## What Changes

- Add a Daily News option to the application navigation.
- Generate a user-specific daily digest from articles published or ingested since the user's last successful digest, or from the past 24 hours when no previous digest exists.
- Run digest generation daily at each user's configured local time, defaulting to 08:00 in Europe/Amsterdam.
- Add user-specific Daily News settings for enablement, generation time, timezone, extra editorial instructions, strict owner-scoped access rules, and an enforceable one-settings-record-per-user invariant.
- Generate a newspaper-like structured Markdown digest using existing entry titles, sources, summaries, takeaways, dates, and effective star ratings.
- Organize the digest with the most important items first, using stars, recency, source context, AI-detected significance, breaking/developing signals, and the user's extra instructions.
- Include a dedicated breaking/developing section when relevant.
- Include a concise "You May Also Find This Interesting" section for lower-rated but potentially useful articles when relevant.
- Link referenced articles to KnowledgeHub entry cards so the user can inspect the article inside the app before opening the original source.
- Allow manual generation and regeneration through server-side routes with atomic duplicate-active-job handling; regeneration overwrites the current digest version for the selected period while preserving that period.
- Retain previous digests indefinitely and provide a paginated way to browse them.
- Create an explicit "No articles today" digest when there are no candidate entries.
- Surface pending or failed digest states when generation cannot complete, such as missing AI configuration or LLM failure, using sanitized user-safe error messages.
- Bound digest prompt size deterministically and record when only a subset of candidates was sent to the LLM.
- Render digest Markdown through a strict sanitizer, render KnowledgeHub entry references only from validated structured IDs, and construct prompts so article/user text is treated as untrusted data rather than instructions.

## Capabilities

### New Capabilities
- `daily-news`: User-specific scheduled and manual Daily News digest generation, storage, browsing, rendering, settings, and KnowledgeHub entry references.

### Modified Capabilities
- `feed-view`: Add an in-app entry-card modal/deep-link behavior so Daily News references can open the specific KnowledgeHub article card.

## Impact

- Backend collections: new user-owned Daily News digest and settings storage.
- Backend scheduler: new per-user daily scheduling logic based on local time and timezone.
- AI processing: new digest-generation prompt and parser using existing article summaries rather than raw article content.
- Routes/API: endpoints or collection operations for manual generation/regeneration and digest retrieval.
- Frontend navigation and pages: Daily News page, archive browsing, settings controls, Markdown rendering, and entry-card modal behavior.
- Tests: scheduler timing, digest window selection, AI prompt behavior, settings persistence, failure states, archive pagination, and UI logic.
