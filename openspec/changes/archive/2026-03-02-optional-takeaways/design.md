## Context

KnowledgeHub currently produces flat 2-4 sentence summaries via a single LLM call that returns `{"summary": "...", "stars": N}`. The summary is stored as a text field on the `entries` collection and rendered in EntryCard. For longer or more complex articles, users lack a quick-scan option — they must read the full summary paragraph to decide if the content is worth opening.

## Goals / Non-Goals

**Goals:**
- Add optional key takeaways (up to 5 bullet points) to the summarization output
- Let the LLM decide whether takeaways add value — omit them for short/simple articles
- Display takeaways in the feed UI when present
- Maintain the single-LLM-call-per-entry cost model

**Non-Goals:**
- Changing the summary itself (length, format, quality)
- Adding takeaways to the ScoreOnly path (fragments are already short)
- Retroactively generating takeaways for existing entries
- Making takeaways user-editable

## Decisions

### 1. Optional JSON field with explicit LLM instruction

The prompt will instruct the LLM to include a `"takeaways"` array only when the article is long or complex enough that bullet points add value beyond the summary. The JSON contract becomes `{"summary": "...", "stars": N}` or `{"summary": "...", "stars": N, "takeaways": ["...", "..."]}`.

**Alternatives considered:**
- Always require takeaways → wasteful for short articles, adds noise
- Separate LLM call for takeaways → doubles cost, violates single-call design
- Content-length threshold in Go code → the LLM is better positioned to judge whether takeaways add value

### 2. Store takeaways as JSON array field

The `entries` collection gets a `takeaways` JSON field (nullable). This keeps each takeaway as a discrete string, making frontend rendering trivial (`{#each takeaways as t}`) and avoids parsing markdown bullets.

**Alternatives considered:**
- Append bullets to the summary text field → harder to style separately, can't toggle display
- Separate `takeaways` collection → over-engineered for a simple array

### 3. Graceful parsing — treat missing takeaways as empty

`parseSummaryResult` will use a `[]string` pointer or check for the key. If `takeaways` is absent or null in the JSON, the field is stored as null/empty. Existing entries and responses without takeaways continue to work unchanged.

## Risks / Trade-offs

- **LLM compliance** → Some models may always or never include takeaways despite instructions. Mitigation: the field is optional, so both behaviors produce valid output. We can tune the prompt if a specific model misbehaves.
- **Prompt length increase** → Adding takeaway instructions slightly increases the system prompt. Mitigation: negligible — a few extra sentences.
- **UI clutter on mobile** → Takeaways add vertical space to cards. Mitigation: render them in a compact style (small text, tight line-height) and only when present.
