## Why

The River redesign v3 mockup has interaction gaps beyond those addressed in v3:

1. **Sidebar source filters are single-select** — users can pick one source or all, but not a subset (e.g., "Rust Blog + InfoQ"). This forces an all-or-nothing filter that doesn't match real triage behavior. *(Addressed in v3)*
2. **Low-priority row click targets conflict** — the entire row is a click target for expand/collapse, but users also expect to click the title to open the article. These two intents share one gesture. *(Addressed in v3)*
3. **No batch collapse for low-priority rows** — when multiple rows are expanded during triage, there's no quick reset. Users must click each row individually. *(Addressed in v3)*
4. **Missing version text and GitHub link** — the production app shows a version string and GitHub icon in the nav bar. The mockup omits these, creating a gap between design and production. *(Addressed in v3)*
5. **Only low-priority cards have expand/collapse** — Featured (5★), High Priority (4★), and Worth a Look (3★) cards have fixed disclosure levels. Users can't collapse a featured card to reclaim space or expand a 3★ row to see the summary without opening the full article. *(New in v4)*
6. **Card body is not a click target** — clicking the card only works if you hit the title `<a>` link precisely. On mobile this is a small target. *(New in v4)*
7. **No section-level batch controls on Featured and High Priority** — v4 added "Collapse all" to Worth a Look and Low Priority, but Featured and HP had no section-level controls. No section had "Expand all". Users who collapse several cards must re-expand them one by one. *(New in v5)*

## What Changes

- **Source filter sidebar**: change from single-select (radio) to multi-select toggles. Each source gets a checkbox indicator. A "Clear" link in the header resets all filters. The topbar shows one dismissable chip per active source. *(v3)*
- **Card interactions — expand/collapse on ALL tiers**: Every card tier (Featured, High Priority, Worth a Look, Low Priority) gets a dedicated ▸/▾ expand/collapse button. Featured and HP default to expanded; Worth a Look and LP default to collapsed. *(v4 — extends v3's LP-only approach)*
- **Card interactions — card-level click opens article**: Clicking anywhere on a card (except the expand button or action buttons) opens the article. Buttons use `event.stopPropagation()` to prevent interference. *(v4)*
- **Section-level batch controls on EVERY section**: All four section headers (Featured, High Priority, Worth a Look, Low Priority) show both "Expand all" and "Collapse all" buttons. Buttons use progressive disclosure — visible only when they have work to do. *(v5 — replaces v4's WaL/LP-only collapse-all)*
- **Version + GitHub**: the sidebar logo area shows version text (e.g., `v0.4.2`) and a small GitHub SVG icon linking to the repository, matching the existing `Nav.svelte` pattern. *(v3)*

## Capabilities

### Modified Capabilities
- `feed-view`: Update sidebar source filter from single-select to multi-select. Update topbar source indicator from single chip to multi-chip. Add section-level "Expand all" and "Collapse all" batch controls to all four sections (Featured, High Priority, Worth a Look, Low Priority).
- `entry-card`: Add ▸/▾ expand/collapse button to all four card tiers. Make entire card a click target for opening article. Featured/HP cards collapsible; WaL/LP cards expandable.
- `navigation`: Add version text and GitHub icon link to sidebar/nav header.

## Impact

- **Frontend (`ui/`)**: Changes to the feed view (`+page.svelte`), sidebar source filter logic, `EntryCard.svelte` (expand/collapse on all tiers, card-level click handler), and `Nav.svelte` (version + GitHub already exists, but River sidebar layout needs to match).
- **Mockup (`designs/mockup.html`)**: Updated to v5 with section-level batch controls on every section.
- **Proposal (`designs/proposal.md`)**: Extended with v5 refinement section documenting section-level batch controls on all sections.
- **Backend**: No changes required — all changes are purely frontend/design.
