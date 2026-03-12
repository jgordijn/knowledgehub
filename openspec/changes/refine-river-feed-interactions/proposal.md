## Why

The River redesign v2 mockup has four interaction/UX gaps:

1. **Sidebar source filters are single-select** — users can pick one source or all, but not a subset (e.g., "Rust Blog + InfoQ"). This forces an all-or-nothing filter that doesn't match real triage behavior.
2. **Low-priority row click targets conflict** — the entire row is a click target for expand/collapse, but users also expect to click the title to open the article. These two intents share one gesture.
3. **No batch collapse for low-priority rows** — when multiple rows are expanded during triage, there's no quick reset. Users must click each row individually.
4. **Missing version text and GitHub link** — the production app shows a version string and GitHub icon in the nav bar. The mockup omits these, creating a gap between design and production.

## What Changes

- **Source filter sidebar**: change from single-select (radio) to multi-select toggles. Each source gets a checkbox indicator. A "Clear" link in the header resets all filters. The topbar shows one dismissable chip per active source.
- **Card interactions**: article titles become `<a>` links across all tiers. Low-priority rows get a dedicated ▸/▾ button for expand/collapse, separate from the title link. Row background is no longer a click target.
- **Collapse all**: a "Collapse all" button appears in the Low Priority section header when ≥1 row is expanded. It hides automatically when all rows are collapsed.
- **Version + GitHub**: the sidebar logo area shows version text (e.g., `v0.4.2`) and a small GitHub SVG icon linking to the repository, matching the existing `Nav.svelte` pattern.

## Capabilities

### Modified Capabilities
- `feed-view`: Update sidebar source filter from single-select to multi-select. Update topbar source indicator from single chip to multi-chip. Add collapse-all control to low-priority section.
- `entry-card`: Make titles clickable links. Separate open-article from expand/collapse on low-priority rows with dedicated expand button.
- `navigation`: Add version text and GitHub icon link to sidebar/nav header.

## Impact

- **Frontend (`ui/`)**: Changes to the feed view (`+page.svelte`), sidebar source filter logic, `EntryCard.svelte` (title as link, expand button), and `Nav.svelte` (version + GitHub already exists, but River sidebar layout needs to match).
- **Mockup (`designs/mockup.html`)**: Updated to v3 with all four changes demonstrated.
- **Proposal (`designs/proposal.md`)**: Extended with v3 refinement section documenting the decisions.
- **Backend**: No changes required — all changes are purely frontend/design.
