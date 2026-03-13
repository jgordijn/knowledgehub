## Why

The River redesign has been fully prototyped in a standalone HTML mockup (`designs/mockup.html`) and refined through five iterations (v1–v5). The mockup demonstrates a tiered card hierarchy, multi-select source filtering, universal expand/collapse, card-level click-to-open, and section-level batch controls. None of these features exist in the production SvelteKit application — the current feed view treats every entry identically as a uniform card. The mockup is validated; now it needs to be implemented in the real app.

## What Changes

- **Tiered card rendering**: `EntryCard.svelte` renders four distinct card tiers based on effective star rating — Featured (5★) as a full-width accent card, High Priority (4★) as a medium card with summary, Worth a Look (3★) as a compact single-line row, and Low Priority (1–2★) as a minimal muted row. Currently all entries render identically.
- **Expand/collapse on every card tier**: Each card tier gets a dedicated ▸/▾ toggle button. Featured and HP default to expanded (detail visible); WaL and LP default to collapsed (detail hidden). Toggling reveals/hides the summary, actions, and metadata without navigating away.
- **Card-level click opens article**: Clicking anywhere on a card (except interactive elements like buttons, links, star rating) opens the article in a new tab and marks it as read. This replaces the current small "↗" link as the primary open gesture.
- **Section headers with batch controls**: The feed view groups entries by tier with section headers ("Featured", "High Priority", "Worth a Look", "Low Priority"). Each header has "Expand all" and "Collapse all" buttons using progressive disclosure — they only appear when they have work to do.
- **Multi-select source filtering in sidebar**: Replace the current collapsible single-select source pill bar with a sidebar source list where each source is an independent toggle. When no sources are selected, all are shown. Active sources display as dismissable chips in the topbar. A "Clear" link resets all filters.
- **Sidebar layout**: Move from the current top-nav bar layout to a sidebar-based layout for desktop, matching the mockup. The sidebar contains navigation links (Feed, Sources, Settings), source filter list, version text, and GitHub icon. On mobile, the sidebar becomes a slide-out drawer.
- **Bookmark filter tab**: Rename current "Read Later" to match the mockup's "Saved" terminology and integrate bookmarked count into the tab bar.

## Capabilities

### New Capabilities
- `sidebar-navigation`: Sidebar-based app shell with navigation links, source filter list, version + GitHub link, mobile slide-out drawer, and theme toggle. Replaces the current top-nav layout.
- `tiered-cards`: Star-based card tier rendering (Featured/HP/WaL/LP) with per-tier layout, expand/collapse toggle, and card-level click-to-open behavior.
- `section-batch-controls`: Section headers that group entries by tier with "Expand all" / "Collapse all" batch buttons using progressive disclosure.
- `multi-select-source-filter`: Multi-select source filter toggles in the sidebar with topbar chip indicators and "Clear" link. Replaces single-select resource filter.

### Modified Capabilities
- `feed-view`: The feed view's entry rendering, filtering, and layout fundamentally change — entries are grouped by tier instead of a flat list, source filtering uses multi-select instead of single-select, and the page layout shifts from top-nav to sidebar.

## Impact

- **Frontend (`ui/`)**: Major changes across multiple files:
  - `+layout.svelte` — New app shell with sidebar instead of top `<Nav>` bar
  - `+page.svelte` — Tiered entry grouping, section headers with batch controls, multi-select source filter state, topbar redesign
  - `EntryCard.svelte` — Replaced with tier-aware rendering (four distinct visual treatments), expand/collapse state, card-level click handler
  - `Nav.svelte` — Likely replaced by new sidebar component, or heavily reworked to serve as the sidebar
  - New components may be needed: `Sidebar.svelte`, `SectionHeader.svelte`, `SourceFilter.svelte`
- **Backend**: No changes required — all changes are purely frontend/UI
- **CSS/Tailwind**: Significant new styling for tiered cards, sidebar layout, expand/collapse animations, source filter toggles, mobile responsive breakpoints
- **Dependencies**: No new dependencies — uses existing SvelteKit 5 + Tailwind CSS
