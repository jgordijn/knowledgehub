## 1. Database Schema

- [ ] 1.1 Add `tags` JSON field (nullable) to the entries collection in `collections.go`
- [ ] 1.2 Add migration logic to update existing collection schema (PocketBase auto-migration on startup)

## 2. AI Processing Pipeline

- [ ] 2.1 Update the LLM prompt in the processing engine to request a `tags` array (1-5 lowercase kebab-case topic strings) alongside summary, stars, and takeaways
- [ ] 2.2 Parse the `tags` field from the LLM JSON response and normalize values (lowercase, replace spaces with hyphens)
- [ ] 2.3 Store parsed tags in the entry's `tags` field during processing
- [ ] 2.4 Add test: verify LLM response parsing handles tags field correctly (present, absent, malformed)

## 3. Backfill Command

- [ ] 3.1 Create a backfill endpoint or CLI command that queries entries where `tags` is null and `summary` is not null
- [ ] 3.2 For each untagged entry, send title + summary to the LLM with a tag-extraction-only prompt (no need to regenerate summary/stars)
- [ ] 3.3 Store extracted tags, skip entries that already have tags
- [ ] 3.4 Add rate limiting / batch size control to avoid overwhelming OpenRouter

## 4. Frontend: Tag Display on Entry Cards

- [ ] 4.1 Add tag chip rendering to `EntryCard.svelte` — small, muted pills below the summary/takeaways section
- [ ] 4.2 Only render the tag section when `entry.tags` is a non-empty array
- [ ] 4.3 Style tag chips to be compact and visually secondary to the card content

## 5. Frontend: Tag Filter

- [ ] 5.1 Add a `selectedTags` state variable (string array) and a `tagFilter` collapsible section on `+page.svelte`, similar to the resource filter
- [ ] 5.2 Compute tag frequencies from currently loaded entries and display the top 20 tags ordered by frequency
- [ ] 5.3 Tags are multi-select pills — clicking toggles selection
- [ ] 5.4 Wire tag filter into entry filtering: when tags are selected, filter entries to those containing at least one selected tag (OR logic). This can be client-side since tags are already loaded with entries.
- [ ] 5.5 Tag filter composes with search, star, read status, and resource filters

## 6. Testing & Verification

- [ ] 6.1 Run Go test suite (`go test ./internal/... -count=1`) and verify all pass
- [ ] 6.2 Manual test: process a new article, verify tags appear on the entry card
- [ ] 6.3 Manual test: run backfill on existing entries, verify tags are populated
- [ ] 6.4 Manual test: select tags in the filter, verify entries filter correctly
- [ ] 6.5 Manual test: combine tag filter with search and star filter, verify composition
