## Why

Articles currently have no categorization beyond which source they come from. When browsing the feed, there is no way to find all articles about a topic (e.g., "distributed systems" or "Go") across different sources. The AI processing pipeline already reads each article to generate summaries and star ratings — adding tag extraction to the same LLM call is low marginal cost and enables powerful topic-based filtering.

## What Changes

- Add a `tags` JSON field to the entries collection, storing an array of lowercase tag strings
- Extend the existing AI processing prompt to also extract 1-5 topic tags per article alongside the summary and star rating (single LLM call, no extra API cost)
- Add a tag filter to the feed view UI that allows multi-select filtering by tag
- Tags are displayed on entry cards as small chips
- Popular tags are surfaced in the filter UI, ordered by frequency

## Capabilities

### New Capabilities
- `tag-filtering`: Multi-select tag filter on the feed view. Shows the most frequently used tags. Selecting one or more tags filters entries to those containing at least one of the selected tags. Composes with all other filters.

### Modified Capabilities
- `ai-processing`: Extend the summarization+scoring prompt to also return a `tags` array of 1-5 lowercase topic strings. Tags capture the article's main subjects (e.g., "go", "distributed-systems", "performance", "llm"). Tags are stored in the entry's `tags` JSON field.
- `feed-view`: Display tags as small chips on entry cards. Add a tag filter section (collapsible, like the current source filter) showing popular tags as selectable pills.

## Impact

- **Backend**: Modify the AI processing prompt in the Go engine to request tags in the LLM response JSON. Parse and store tags in the new field. Add a one-time backfill endpoint or CLI command to tag existing untagged entries.
- **Frontend**: New tag chips on `EntryCard.svelte`. New tag filter section on `+page.svelte`. PocketBase filter using `tags ~ '"tagname"'` for JSON array contains.
- **Database**: Add `tags` JSON field to the entries collection (nullable, defaults to null for existing entries).

## Non-goals

- User-defined custom tags or manual tagging (AI-only for now)
- Tag taxonomy or hierarchy (flat list is sufficient)
- Tag management UI (tags are derived, not curated)
- Reprocessing all existing articles automatically on deploy (backfill is opt-in via CLI)
