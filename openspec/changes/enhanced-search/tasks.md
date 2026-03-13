## 1. Search Bar Component

- [ ] 1.1 Add a search input field to `+page.svelte` above the existing filter controls, with placeholder text "Search articles..."
- [ ] 1.2 Add `searchQuery` state variable and a 300ms debounce mechanism (use a `setTimeout`-based debounce or a reactive `$effect` with delay)
- [ ] 1.3 Wire the debounced search query into `loadEntries()` — when non-empty, add PocketBase filter clauses: `(title ~ '{query}' || summary ~ '{query}' || resource.name ~ '{query}')`
- [ ] 1.4 Trigger `loadEntries()` reactively when `searchQuery` changes (add to the existing `$effect` alongside `readFilter` and `resourceFilter`)

## 2. Multi-Select Resource Filter

- [ ] 2.1 Change `resourceFilter` state from `string` (single ID) to `string[]` (array of IDs)
- [ ] 2.2 Update resource pill buttons to toggle selection (add/remove from array) instead of single-select
- [ ] 2.3 Update `loadEntries()` filter construction — when multiple resources selected, build an OR filter: `(resource = 'id1' || resource = 'id2' || ...)`
- [ ] 2.4 Add dismissible chip display above the entry list showing selected resource names with an X button to deselect each
- [ ] 2.5 Update the "All" button to clear the array, and the collapsible header to show count of selected sources

## 3. Empty State & UX

- [ ] 3.1 Add a search-specific empty state message: "No entries match your search" when search is active and results are empty
- [ ] 3.2 Add a clear-search affordance (X button inside the search input or a "Clear filters" link)

## 4. Testing & Verification

- [ ] 4.1 Manual test: type a search query, verify entries filter by title match
- [ ] 4.2 Manual test: search for a resource name (e.g., "Yegge"), verify entries from that source appear
- [ ] 4.3 Manual test: combine search with star filter and read filter, verify all compose correctly
- [ ] 4.4 Manual test: select multiple resources, verify only entries from those sources appear
- [ ] 4.5 Manual test: dismiss a resource chip, verify filter updates
- [ ] 4.6 Manual test: verify debounce — rapid typing should not cause excessive loading flicker
