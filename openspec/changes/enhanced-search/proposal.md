## Why

Finding a specific article in the feed is difficult. There is no text search, so the only way to narrow results is by toggling read status, star rating, or selecting a single source from a collapsible pill list. If you remember an article was by "Yegge" but don't remember which source it came from, you have to manually scan through entries or click through sources one by one. As the article collection grows, this becomes increasingly painful.

## What Changes

- Add a search bar at the top of the feed view that filters entries by matching against title, summary, and resource name
- The search is server-side using PocketBase filter operators, so it works within the existing 200-item page and composes with all other active filters
- The resource filter panel switches from single-select to multi-select, so users can view articles from several sources at once
- Active filters (search query, selected sources) are shown as dismissible chips for clarity

## Capabilities

### New Capabilities
- `article-search`: Full-text search bar on the feed view that filters entries by title, summary, and expanded resource name. Composes with existing read status, star, and resource filters.

### Modified Capabilities
- `feed-view`: Add the search bar above existing filter controls. Change resource filter from single-select to multi-select with dismissible chips. Add empty-state messaging when search returns no results.

## Impact

- **Frontend**: New search input component on `+page.svelte`. Modified resource filter from radio-style to checkbox-style pills. Dismissible filter chip display. The `loadEntries` function gains a search term parameter that adds PocketBase `~` filter clauses.
- **Backend**: No backend changes required. PocketBase already supports the `~` (LIKE) operator and relation field filtering needed for this feature.
- **Database**: No schema changes. Searching uses existing `title`, `summary` fields and the expanded `resource.name` field.

## Non-goals

- Dedicated search results page or search history
- Fuzzy/typo-tolerant search (PocketBase `~` is substring match, which is sufficient)
- Pagination beyond the current 200-item limit (separate concern)
- Search within article body/raw_content (too heavy, summary is sufficient)
