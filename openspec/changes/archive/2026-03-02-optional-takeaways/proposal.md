## Why

Article summaries are flat 2-4 sentence blobs. For longer or more complex articles, users must read the entire summary to decide if the content is worth clicking through. A scannable list of key takeaways — inspired by the tldr skill's format — would let users triage faster, especially on mobile. Not every article needs takeaways though; short articles are already well-served by the summary alone.

## What Changes

- The LLM prompt is updated to optionally produce key takeaways (up to 5 bullet points) alongside the summary. Takeaways are produced only when the article is complex or long enough that the summary alone doesn't cover the key points.
- The `entries` collection gains a `takeaways` field (JSON array of strings, nullable).
- The `SummaryResult` struct and JSON parsing are extended to handle the optional `takeaways` field.
- The feed UI entry cards conditionally render takeaways as a bullet list below the summary when present.
- The ScoreOnly path is unaffected — fragments don't get takeaways.

## Capabilities

### New Capabilities

_None — this extends existing capabilities._

### Modified Capabilities

- `ai-processing`: The summarization prompt and result format change to optionally include takeaways. The combined LLM call contract expands from `{summary, stars}` to `{summary, stars, takeaways?}`.
- `feed-view`: Entry cards conditionally display takeaways below the summary when the entry has them.

## Impact

- **Backend**: `internal/ai/summarizer.go` — prompt text, `SummaryResult` struct, JSON parsing
- **Backend**: `cmd/knowledgehub/collections.go` — add `takeaways` field to `entries` collection
- **Frontend**: `ui/src/lib/components/EntryCard.svelte` — render takeaways list
- **Tests**: Update AI summarizer tests for new JSON shape; existing tests must still pass when takeaways are absent
- **No breaking changes**: The field is optional/nullable. Existing entries without takeaways continue to work unchanged.
