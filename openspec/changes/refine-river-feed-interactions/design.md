## Context

The River redesign mockup (v2) established the core tiered-card feed layout, sidebar source filtering, and expandable low-priority rows. Review feedback identified four interaction gaps: single-select source filters are too restrictive, row-level click handling conflicts with article navigation, there's no batch collapse, and the mockup is missing production chrome (version, GitHub link).

This change refines the mockup and proposal to address all four gaps. It is design-only — no app source code is modified.

## Goals / Non-Goals

**Goals:**
- Multi-select source filter toggles in the sidebar with clear visual state
- Conflict-free click targets: title links open articles, dedicated buttons expand/collapse
- "Collapse all" affordance for the low-priority section
- Version text + GitHub icon in the sidebar matching the production `Nav.svelte`

**Non-Goals:**
- Implementing these changes in the SvelteKit app (that's a separate change)
- Changing the River tier hierarchy or star-based card prominence
- Adding new features beyond what the mockup already demonstrates

## Decisions

### 1. Multi-select toggles with checkbox indicators

**Decision:** Replace the single-select (radio-style) sidebar source list with independent toggles. Each source item gets a 14×14px checkbox square that fills blue with a ✓ when active.

**Rationale:** Single-select forced an all-or-nothing filter. Multi-select is strictly more expressive and matches familiar patterns (Gmail labels, GitHub filters). The checkbox provides unambiguous state: checked = active, unchecked = inactive.

**Behavior:**
- When no sources are selected → all sources shown (unfiltered, default state)
- When ≥1 source selected → only those sources' entries appear
- "Clear" link in section header removes all selections
- Topbar shows one blue chip per active source, each with ✕ dismiss

### 2. Separated click targets for open vs expand

**Decision:** Article titles are `<a>` links everywhere (Featured, High Priority, Worth a Look, Low Priority). Low-priority rows use a dedicated ▸/▾ `<button>` for expand/collapse instead of making the entire row clickable.

**Rationale:** When the whole row was a click target, clicking the title to read the article also toggled expand — a frustrating conflict. Separating these into distinct elements follows standard accessibility patterns. The expand button is small and positioned at the right edge (secondary action). The title is front-and-center (primary action).

**Implementation notes for future app work:**
- `EntryCard.svelte` title becomes an `<a>` tag with `href={entry.url}` and `target="_blank"`
- Low-priority variant adds a chevron `<button>` with `onclick` that toggles a detail `<div>`
- `event.stopPropagation()` on the title link prevents any parent click handlers from firing

### 3. Collapse-all is progressive disclosure

**Decision:** The "Collapse all" button appears in the Low Priority section header only when ≥1 row is expanded. It hides when all rows are collapsed.

**Rationale:** Showing a collapse button when nothing is expanded is confusing. Progressive disclosure keeps the default state clean. The button uses the same styling as the expand chevrons — small, muted, unobtrusive until needed.

### 4. Version + GitHub match production chrome

**Decision:** Show version text (e.g., `v0.4.2`) and a 15×15px GitHub SVG icon in the sidebar logo area, matching the existing `Nav.svelte` layout.

**Rationale:** The mockup should reflect actual production UI elements. Version text helps users verify their release. The GitHub icon is a subtle, discoverable link — not prominent enough to distract, but available for anyone looking for the repo.

**Implementation notes:** The production app fetches version from `GET /api/version`. The GitHub SVG path is identical to the one in `Nav.svelte` line 40.

## Technical Details

All changes are to design/mockup files only:

| File | Changes |
|------|---------|
| `designs/mockup.html` | Multi-select source toggle JS/CSS, separated click targets, collapse-all button, version+GitHub in sidebar |
| `designs/proposal.md` | New "Refinement v3" section documenting all four decisions, updated quick-reference table |
| `openspec/changes/refine-river-feed-interactions/*` | Proposal, design, and tasks capturing these requirements |
